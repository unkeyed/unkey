import { mysqlDrizzle, schema, sql } from "@unkey/db";
import { newId } from "@unkey/id";
import mysql from "mysql2/promise";

async function main() {
  if (!process.env.DRIZZLE_DATABASE_URL) {
    throw new Error("DRIZZLE_DATABASE_URL is not set");
  }

  const conn = await mysql.createConnection(process.env.DRIZZLE_DATABASE_URL);
  await conn.ping();
  const db = mysqlDrizzle(conn, { schema, mode: "default" });

  // 1. Get all projects with their environments and apps
  const projects = await db.query.projects.findMany({
    with: {
      apps: true,
      environments: true,
    },
  });
  console.log(`Found ${projects.length} projects`);

  // 2. Create a default app per environment for each project
  const envToAppId: Record<string, string> = {};
  const envToWorkspaceId: Record<string, string> = {};
  let appsCreated = 0;

  for (const project of projects) {
    const prodEnv = project.environments.find((e) => e.slug === "production");

    for (const env of project.environments) {
      envToWorkspaceId[env.id] = env.workspaceId;

      // Check if an app already exists for this environment
      const existing = project.apps.find((a) => a.environmentId === env.id);
      if (existing) {
        envToAppId[env.id] = existing.id;
        continue;
      }

      const appId = newId("app");
      const isProd = env.id === prodEnv?.id;

      await db.insert(schema.apps).values({
        id: appId,
        workspaceId: project.workspaceId,
        projectId: project.id,
        environmentId: env.id,
        name: "Default",
        slug: "default",
        // Only the production app inherits project-level deployment/depot settings
        currentDeploymentId: isProd ? (project.liveDeploymentId ?? null) : null,
        isRolledBack: isProd ? (project.isRolledBack ?? false) : false,
        createdAt: Date.now(),
      });

      envToAppId[env.id] = appId;
      appsCreated++;
      console.log(
        `Created default app ${appId} for env ${env.slug} (${env.id}) in project ${project.id}`,
      );
    }
  }
  console.log(`Apps created: ${appsCreated}`);

  // 4. Copy environment_build_settings -> app_build_settings
  const buildSettings = await db.query.environmentBuildSettings.findMany();
  let buildCopied = 0;
  for (const bs of buildSettings) {
    const appId = envToAppId[bs.environmentId];
    if (!appId) {
      continue;
    }

    await db
      .insert(schema.appBuildSettings)
      .values({
        workspaceId: bs.workspaceId,
        appId,
        environmentId: bs.environmentId,
        dockerfile: bs.dockerfile,
        dockerContext: bs.dockerContext,
        createdAt: bs.createdAt,
        updatedAt: bs.updatedAt,
      })
      .onDuplicateKeyUpdate({
        set: { dockerfile: bs.dockerfile, dockerContext: bs.dockerContext },
      });
    buildCopied++;
  }
  console.log(`Build settings copied: ${buildCopied}`);

  // 5. Copy environment_runtime_settings -> app_runtime_settings
  const runtimeSettings = await db.query.environmentRuntimeSettings.findMany();
  let runtimeCopied = 0;
  for (const rs of runtimeSettings) {
    const appId = envToAppId[rs.environmentId];
    if (!appId) {
      continue;
    }

    await db
      .insert(schema.appRuntimeSettings)
      .values({
        workspaceId: rs.workspaceId,
        appId,
        environmentId: rs.environmentId,
        port: rs.port,
        cpuMillicores: rs.cpuMillicores,
        memoryMib: rs.memoryMib,
        command: rs.command,
        healthcheck: rs.healthcheck,
        regionConfig: rs.regionConfig,
        shutdownSignal: rs.shutdownSignal,
        sentinelConfig: rs.sentinelConfig,
        createdAt: rs.createdAt,
        updatedAt: rs.updatedAt,
      })
      .onDuplicateKeyUpdate({
        set: {
          port: rs.port,
          cpuMillicores: rs.cpuMillicores,
          memoryMib: rs.memoryMib,
          command: rs.command,
          healthcheck: rs.healthcheck,
          regionConfig: rs.regionConfig,
          shutdownSignal: rs.shutdownSignal,
          sentinelConfig: rs.sentinelConfig,
        },
      });
    runtimeCopied++;
  }
  console.log(`Runtime settings copied: ${runtimeCopied}`);

  // 6. Copy environment_variables -> app_environment_variables
  const envVars = await db.query.environmentVariables.findMany();
  let varsCopied = 0;
  for (const ev of envVars) {
    const appId = envToAppId[ev.environmentId];
    if (!appId) {
      continue;
    }

    const id = newId("environmentVariable");
    await db
      .insert(schema.appEnvironmentVariables)
      .values({
        id,
        workspaceId: ev.workspaceId,
        appId,
        environmentId: ev.environmentId,
        key: ev.key,
        value: ev.value,
        type: ev.type,
        description: ev.description,
        deleteProtection: ev.deleteProtection,
        createdAt: ev.createdAt,
        updatedAt: ev.updatedAt,
      })
      .onDuplicateKeyUpdate({
        set: { value: ev.value, type: ev.type, description: ev.description },
      });
    varsCopied++;
  }
  console.log(`Environment variables copied: ${varsCopied}`);

  // 7. Backfill app_id on tables that have environment_id (join through environment)
  console.log("Backfilling app_id on deployments...");
  await db.execute(
    sql`UPDATE deployments d JOIN apps a ON a.environment_id = d.environment_id AND a.slug = 'default' SET d.app_id = a.id WHERE d.app_id = ''`,
  );

  console.log("Backfilling app_id on deployment_steps...");
  await db.execute(
    sql`UPDATE deployment_steps ds JOIN apps a ON a.environment_id = ds.environment_id AND a.slug = 'default' SET ds.app_id = a.id WHERE ds.app_id = ''`,
  );

  console.log("Backfilling app_id on instances...");
  await db.execute(
    sql`UPDATE instances i JOIN deployments d ON d.id = i.deployment_id JOIN apps a ON a.environment_id = d.environment_id AND a.slug = 'default' SET i.app_id = a.id WHERE i.app_id = ''`,
  );

  console.log("Backfilling app_id on frontline_routes...");
  await db.execute(
    sql`UPDATE frontline_routes fr JOIN apps a ON a.environment_id = fr.environment_id AND a.slug = 'default' SET fr.app_id = a.id WHERE fr.app_id = ''`,
  );

  // 8. Backfill app_id on project-scoped tables (use production env's app)
  console.log("Backfilling app_id on cilium_network_policies...");
  await db.execute(
    sql`UPDATE cilium_network_policies cnp JOIN apps a ON a.project_id = cnp.project_id AND a.slug = 'default' JOIN environments e ON e.id = a.environment_id AND e.slug = 'production' SET cnp.app_id = a.id WHERE cnp.app_id = ''`,
  );

  console.log("Backfilling app_id on github_repo_connections...");
  await db.execute(
    sql`UPDATE github_repo_connections grc JOIN apps a ON a.project_id = grc.project_id AND a.slug = 'default' JOIN environments e ON e.id = a.environment_id AND e.slug = 'production' SET grc.app_id = a.id WHERE grc.app_id = ''`,
  );

  console.log("Backfilling app_id on custom_domains...");
  await db.execute(
    sql`UPDATE custom_domains cd JOIN apps a ON a.environment_id = cd.environment_id AND a.slug = 'default' SET cd.app_id = a.id WHERE cd.app_id = ''`,
  );

  console.log("Migration complete!");
  await conn.end();
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});

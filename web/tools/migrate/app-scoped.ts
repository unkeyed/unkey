import { mysqlDrizzle, schema, sql } from "@unkey/db";
import { newId } from "@unkey/id";
import mysql from "mysql2/promise";

async function main() {
  const conn = await mysql.createConnection(
    `mysql://${process.env.DATABASE_USERNAME}:${process.env.DATABASE_PASSWORD}@${process.env.DATABASE_HOST}:3306/unkey?ssl={}`,
  );
  await conn.ping();
  const db = mysqlDrizzle(conn, { schema, mode: "default" });

  // 1. Get all projects with their environments
  const projects = await db.query.projects.findMany({
    with: {
      apps: true,
    },
  });
  console.log(`Found ${projects.length} projects`);

  // 2. Create default app for each project that doesn't have one
  const projectToAppId: Record<string, string> = {};
  let appsCreated = 0;

  for (const project of projects) {
    const existing = project.apps.find((a) => a.slug === "default");
    if (existing) {
      projectToAppId[project.id] = existing.id;
      continue;
    }

    const appId = newId("app");

    await db.insert(schema.apps).values({
      id: appId,
      workspaceId: project.workspaceId,
      projectId: project.id,
      name: "Default",
      slug: "default",
      liveDeploymentId: project.liveDeploymentId ?? null,
      isRolledBack: project.isRolledBack ?? false,
      depotProjectId: project.depotProjectId ?? null,
      createdAt: Date.now(),
    });

    projectToAppId[project.id] = appId;
    appsCreated++;
    console.log(`Created default app ${appId} for project ${project.id}`);
  }
  console.log(`Apps created: ${appsCreated}`);

  // 3. Build environment_id -> app_id map
  const environments = await db.query.environments.findMany({
    columns: { id: true, projectId: true, workspaceId: true },
  });
  const envToAppId: Record<string, string> = {};
  const envToWorkspaceId: Record<string, string> = {};
  for (const env of environments) {
    const appId = projectToAppId[env.projectId];
    if (appId) {
      envToAppId[env.id] = appId;
      envToWorkspaceId[env.id] = env.workspaceId;
    }
  }

  // 4. Copy environment_build_settings -> app_build_settings
  const buildSettings = await db.query.environmentBuildSettings.findMany();
  let buildCopied = 0;
  for (const bs of buildSettings) {
    const appId = envToAppId[bs.environmentId];
    if (!appId) continue;

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
    if (!appId) continue;

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
    if (!appId) continue;

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

  // 7. Backfill app_id on all tables with JOIN UPDATE
  console.log("Backfilling app_id on deployments...");
  await db.execute(
    sql`UPDATE deployments d JOIN apps a ON a.project_id = d.project_id AND a.slug = 'default' SET d.app_id = a.id WHERE d.app_id = ''`,
  );

  console.log("Backfilling app_id on deployment_steps...");
  await db.execute(
    sql`UPDATE deployment_steps ds JOIN apps a ON a.project_id = ds.project_id AND a.slug = 'default' SET ds.app_id = a.id WHERE ds.app_id = ''`,
  );

  console.log("Backfilling app_id on instances...");
  await db.execute(
    sql`UPDATE instances i JOIN apps a ON a.project_id = i.project_id AND a.slug = 'default' SET i.app_id = a.id WHERE i.app_id = ''`,
  );

  console.log("Backfilling app_id on frontline_routes...");
  await db.execute(
    sql`UPDATE frontline_routes fr JOIN apps a ON a.project_id = fr.project_id AND a.slug = 'default' SET fr.app_id = a.id WHERE fr.app_id = ''`,
  );

  console.log("Backfilling app_id on cilium_network_policies...");
  await db.execute(
    sql`UPDATE cilium_network_policies cnp JOIN apps a ON a.project_id = cnp.project_id AND a.slug = 'default' SET cnp.app_id = a.id WHERE cnp.app_id = ''`,
  );

  console.log("Backfilling app_id on github_repo_connections...");
  await db.execute(
    sql`UPDATE github_repo_connections grc JOIN apps a ON a.project_id = grc.project_id AND a.slug = 'default' SET grc.app_id = a.id WHERE grc.app_id = ''`,
  );

  console.log("Migration complete!");
  await conn.end();
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});

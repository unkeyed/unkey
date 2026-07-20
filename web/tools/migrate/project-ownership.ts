import {
  and,
  count,
  createCommentedPool,
  drizzle,
  eq,
  isNotNull,
  isNull,
  ne,
  or,
  schema,
  staticTagsFromEnv,
} from "@unkey/db";
import { newId } from "@unkey/id";

const databaseURL = process.env.DRIZZLE_DATABASE_URL;
if (!databaseURL) {
  throw new Error("DRIZZLE_DATABASE_URL is not set");
}

const pool = createCommentedPool(
  { uri: databaseURL, connectionLimit: 2 },
  staticTagsFromEnv("project-ownership-migration"),
);
const db = drizzle(pool, { schema, mode: "default" });

async function ensureDefaultProjects(): Promise<Map<string, string>> {
  const [workspaceRows, projectRows] = await Promise.all([
    db.select({ id: schema.workspaces.id }).from(schema.workspaces),
    db
      .select({
        id: schema.projects.id,
        workspaceId: schema.projects.workspaceId,
        slug: schema.projects.slug,
      })
      .from(schema.projects)
      .where(eq(schema.projects.slug, "default")),
  ]);

  const conflictingProjects = projectRows.filter((project) => project.slug !== "default");
  if (conflictingProjects.length > 0) {
    throw new Error(
      `${conflictingProjects.length} projects conflict with the reserved exact slug "default"`,
    );
  }

  const defaults = new Map(projectRows.map((project) => [project.workspaceId, project.id]));

  for (const workspace of workspaceRows) {
    if (defaults.has(workspace.id)) {
      continue;
    }

    const projectId = newId("project");
    await db.insert(schema.projects).values({
      id: projectId,
      workspaceId: workspace.id,
      name: "Default",
      slug: "default",
      deleteProtection: true,
      createdAt: Date.now(),
    });
    defaults.set(workspace.id, projectId);
    console.log(`Created default project ${projectId} for workspace ${workspace.id}`);
  }

  return defaults;
}

async function countEmptyOwnership(): Promise<number> {
  const emptyRows = await Promise.all([
    db.select({ invalidCount: count() }).from(schema.apis).where(eq(schema.apis.projectId, "")),
    db
      .select({ invalidCount: count() })
      .from(schema.keyAuth)
      .where(eq(schema.keyAuth.projectId, "")),
    db
      .select({ invalidCount: count() })
      .from(schema.identities)
      .where(eq(schema.identities.projectId, "")),
    db
      .select({ invalidCount: count() })
      .from(schema.permissions)
      .where(eq(schema.permissions.projectId, "")),
    db.select({ invalidCount: count() }).from(schema.roles).where(eq(schema.roles.projectId, "")),
    db
      .select({ invalidCount: count() })
      .from(schema.ratelimitNamespaces)
      .where(eq(schema.ratelimitNamespaces.projectId, "")),
  ]);
  return emptyRows.reduce((total, rows) => total + (rows[0]?.invalidCount ?? 0), 0);
}

async function backfillToConvergence(): Promise<void> {
  for (let sweep = 1; ; sweep++) {
    const defaultProjects = await ensureDefaultProjects();
    const emptyBefore = await countEmptyOwnership();
    if (emptyBefore === 0) {
      return;
    }

    for (const [workspaceId, projectId] of defaultProjects) {
      await db
        .update(schema.apis)
        .set({ projectId })
        .where(and(eq(schema.apis.workspaceId, workspaceId), eq(schema.apis.projectId, "")));
      await db
        .update(schema.keyAuth)
        .set({ projectId })
        .where(and(eq(schema.keyAuth.workspaceId, workspaceId), eq(schema.keyAuth.projectId, "")));
      await db
        .update(schema.identities)
        .set({ projectId })
        .where(
          and(eq(schema.identities.workspaceId, workspaceId), eq(schema.identities.projectId, "")),
        );
      await db
        .update(schema.permissions)
        .set({ projectId })
        .where(
          and(
            eq(schema.permissions.workspaceId, workspaceId),
            eq(schema.permissions.projectId, ""),
          ),
        );
      await db
        .update(schema.roles)
        .set({ projectId })
        .where(and(eq(schema.roles.workspaceId, workspaceId), eq(schema.roles.projectId, "")));
      await db
        .update(schema.ratelimitNamespaces)
        .set({ projectId })
        .where(
          and(
            eq(schema.ratelimitNamespaces.workspaceId, workspaceId),
            eq(schema.ratelimitNamespaces.projectId, ""),
          ),
        );
    }

    console.log(`Completed backfill sweep ${sweep}`);
    const emptyAfter = await countEmptyOwnership();
    if (emptyAfter === 0) {
      return;
    }
    if (emptyAfter >= emptyBefore) {
      throw new Error(
        `Project ownership backfill did not converge: ${emptyAfter} empty rows remain after sweep ${sweep}`,
      );
    }
  }
}

function recordValidation(failures: string[], name: string, invalidCount: number): void {
  console.log(`${name}: ${invalidCount === 0 ? "ok" : `${invalidCount} invalid rows`}`);
  if (invalidCount !== 0) {
    failures.push(`${name} (${invalidCount})`);
  }
}

async function validate(): Promise<void> {
  const failures: string[] = [];
  const [workspaceRows, projectRows] = await Promise.all([
    db.select({ id: schema.workspaces.id }).from(schema.workspaces),
    db
      .select({ workspaceId: schema.projects.workspaceId, slug: schema.projects.slug })
      .from(schema.projects)
      .where(eq(schema.projects.slug, "default")),
  ]);
  recordValidation(
    failures,
    "no project conflicts with the reserved exact default slug",
    projectRows.filter((project) => project.slug !== "default").length,
  );
  const defaultCounts = new Map(workspaceRows.map((workspace) => [workspace.id, 0]));
  for (const project of projectRows) {
    if (project.slug === "default") {
      defaultCounts.set(project.workspaceId, (defaultCounts.get(project.workspaceId) ?? 0) + 1);
    }
  }
  recordValidation(
    failures,
    "every workspace has exactly one default project",
    [...defaultCounts.values()].filter((defaults) => defaults !== 1).length,
  );

  const emptyOwnership = await Promise.all([
    db.select({ invalidCount: count() }).from(schema.apis).where(eq(schema.apis.projectId, "")),
    db
      .select({ invalidCount: count() })
      .from(schema.keyAuth)
      .where(eq(schema.keyAuth.projectId, "")),
    db
      .select({ invalidCount: count() })
      .from(schema.identities)
      .where(eq(schema.identities.projectId, "")),
    db
      .select({ invalidCount: count() })
      .from(schema.permissions)
      .where(eq(schema.permissions.projectId, "")),
    db.select({ invalidCount: count() }).from(schema.roles).where(eq(schema.roles.projectId, "")),
    db
      .select({ invalidCount: count() })
      .from(schema.ratelimitNamespaces)
      .where(eq(schema.ratelimitNamespaces.projectId, "")),
  ]);
  for (const [name, rows] of [
    "APIs",
    "keyspaces",
    "identities",
    "permissions",
    "roles",
    "rate limit namespaces",
  ].map((name, index) => [name, emptyOwnership[index]] as const)) {
    recordValidation(
      failures,
      `no ${name} have empty project ownership`,
      rows?.[0]?.invalidCount ?? 0,
    );
  }

  const resourceOwnership = await Promise.all([
    db
      .select({ invalidCount: count() })
      .from(schema.apis)
      .leftJoin(schema.projects, eq(schema.projects.id, schema.apis.projectId))
      .where(
        or(isNull(schema.projects.id), ne(schema.projects.workspaceId, schema.apis.workspaceId)),
      ),
    db
      .select({ invalidCount: count() })
      .from(schema.keyAuth)
      .leftJoin(schema.projects, eq(schema.projects.id, schema.keyAuth.projectId))
      .where(
        or(isNull(schema.projects.id), ne(schema.projects.workspaceId, schema.keyAuth.workspaceId)),
      ),
    db
      .select({ invalidCount: count() })
      .from(schema.identities)
      .leftJoin(schema.projects, eq(schema.projects.id, schema.identities.projectId))
      .where(
        or(
          isNull(schema.projects.id),
          ne(schema.projects.workspaceId, schema.identities.workspaceId),
        ),
      ),
    db
      .select({ invalidCount: count() })
      .from(schema.permissions)
      .leftJoin(schema.projects, eq(schema.projects.id, schema.permissions.projectId))
      .where(
        or(
          isNull(schema.projects.id),
          ne(schema.projects.workspaceId, schema.permissions.workspaceId),
        ),
      ),
    db
      .select({ invalidCount: count() })
      .from(schema.roles)
      .leftJoin(schema.projects, eq(schema.projects.id, schema.roles.projectId))
      .where(
        or(isNull(schema.projects.id), ne(schema.projects.workspaceId, schema.roles.workspaceId)),
      ),
    db
      .select({ invalidCount: count() })
      .from(schema.ratelimitNamespaces)
      .leftJoin(schema.projects, eq(schema.projects.id, schema.ratelimitNamespaces.projectId))
      .where(
        or(
          isNull(schema.projects.id),
          ne(schema.projects.workspaceId, schema.ratelimitNamespaces.workspaceId),
        ),
      ),
  ]);
  for (const [name, rows] of [
    "API",
    "keyspace",
    "identity",
    "permission",
    "role",
    "rate limit namespace",
  ].map((name, index) => [name, resourceOwnership[index]] as const)) {
    recordValidation(
      failures,
      `every ${name} project belongs to its workspace`,
      rows?.[0]?.invalidCount ?? 0,
    );
  }

  const [
    apiKeyspaces,
    keyIdentities,
    keyPermissions,
    keyRoles,
    rolePermissions,
    namespaceOverrides,
  ] = await Promise.all([
    db
      .select({ invalidCount: count() })
      .from(schema.apis)
      .leftJoin(schema.keyAuth, eq(schema.keyAuth.id, schema.apis.keyAuthId))
      .where(
        and(
          isNotNull(schema.apis.keyAuthId),
          or(
            isNull(schema.keyAuth.id),
            ne(schema.keyAuth.workspaceId, schema.apis.workspaceId),
            ne(schema.keyAuth.projectId, schema.apis.projectId),
          ),
        ),
      ),
    db
      .select({ invalidCount: count() })
      .from(schema.keys)
      .leftJoin(schema.keyAuth, eq(schema.keyAuth.id, schema.keys.keyAuthId))
      .leftJoin(schema.identities, eq(schema.identities.id, schema.keys.identityId))
      .where(
        and(
          isNotNull(schema.keys.identityId),
          or(
            isNull(schema.keyAuth.id),
            isNull(schema.identities.id),
            ne(schema.keys.workspaceId, schema.keyAuth.workspaceId),
            ne(schema.keys.workspaceId, schema.identities.workspaceId),
            ne(schema.keyAuth.projectId, schema.identities.projectId),
          ),
        ),
      ),
    db
      .select({ invalidCount: count() })
      .from(schema.keysPermissions)
      .leftJoin(schema.keys, eq(schema.keys.id, schema.keysPermissions.keyId))
      .leftJoin(schema.keyAuth, eq(schema.keyAuth.id, schema.keys.keyAuthId))
      .leftJoin(schema.permissions, eq(schema.permissions.id, schema.keysPermissions.permissionId))
      .where(
        or(
          isNull(schema.keys.id),
          isNull(schema.keyAuth.id),
          isNull(schema.permissions.id),
          ne(schema.keysPermissions.workspaceId, schema.keys.workspaceId),
          ne(schema.keys.workspaceId, schema.keyAuth.workspaceId),
          ne(schema.keys.workspaceId, schema.permissions.workspaceId),
          ne(schema.keyAuth.projectId, schema.permissions.projectId),
        ),
      ),
    db
      .select({ invalidCount: count() })
      .from(schema.keysRoles)
      .leftJoin(schema.keys, eq(schema.keys.id, schema.keysRoles.keyId))
      .leftJoin(schema.keyAuth, eq(schema.keyAuth.id, schema.keys.keyAuthId))
      .leftJoin(schema.roles, eq(schema.roles.id, schema.keysRoles.roleId))
      .where(
        or(
          isNull(schema.keys.id),
          isNull(schema.keyAuth.id),
          isNull(schema.roles.id),
          ne(schema.keysRoles.workspaceId, schema.keys.workspaceId),
          ne(schema.keys.workspaceId, schema.keyAuth.workspaceId),
          ne(schema.keys.workspaceId, schema.roles.workspaceId),
          ne(schema.keyAuth.projectId, schema.roles.projectId),
        ),
      ),
    db
      .select({ invalidCount: count() })
      .from(schema.rolesPermissions)
      .leftJoin(schema.roles, eq(schema.roles.id, schema.rolesPermissions.roleId))
      .leftJoin(schema.permissions, eq(schema.permissions.id, schema.rolesPermissions.permissionId))
      .where(
        or(
          isNull(schema.roles.id),
          isNull(schema.permissions.id),
          ne(schema.rolesPermissions.workspaceId, schema.roles.workspaceId),
          ne(schema.roles.workspaceId, schema.permissions.workspaceId),
          ne(schema.roles.projectId, schema.permissions.projectId),
        ),
      ),
    db
      .select({ invalidCount: count() })
      .from(schema.ratelimitOverrides)
      .leftJoin(
        schema.ratelimitNamespaces,
        eq(schema.ratelimitNamespaces.id, schema.ratelimitOverrides.namespaceId),
      )
      .where(
        or(
          isNull(schema.ratelimitNamespaces.id),
          ne(schema.ratelimitOverrides.workspaceId, schema.ratelimitNamespaces.workspaceId),
        ),
      ),
  ]);
  recordValidation(
    failures,
    "linked APIs and keyspaces have consistent project ownership",
    apiKeyspaces[0]?.invalidCount ?? 0,
  );
  recordValidation(
    failures,
    "linked keys and identities have consistent project ownership",
    keyIdentities[0]?.invalidCount ?? 0,
  );
  recordValidation(
    failures,
    "linked keys and permissions have consistent project ownership",
    keyPermissions[0]?.invalidCount ?? 0,
  );
  recordValidation(
    failures,
    "linked keys and roles have consistent project ownership",
    keyRoles[0]?.invalidCount ?? 0,
  );
  recordValidation(
    failures,
    "linked roles and permissions have consistent project ownership",
    rolePermissions[0]?.invalidCount ?? 0,
  );
  recordValidation(
    failures,
    "rate limit overrides belong to their namespace workspace",
    namespaceOverrides[0]?.invalidCount ?? 0,
  );

  if (failures.length > 0) {
    throw new Error(`Project ownership validation failed: ${failures.join(", ")}`);
  }
}

async function main(): Promise<void> {
  try {
    await backfillToConvergence();
    await validate();
    console.log("Project ownership migration complete");
  } finally {
    await pool.end();
  }
}

main().catch((error: unknown) => {
  console.error(error);
  process.exitCode = 1;
});

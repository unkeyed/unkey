import { db, desc, eq } from "@/lib/db";
import { workspaceProcedure } from "@/lib/trpc/trpc";
import {
  apps,
  deployments,
  environments,
  identities,
  permissions,
  projects,
  ratelimitNamespaces,
  ratelimitOverrides,
  roles,
} from "@unkey/db/src/schema";

const limit = 200;

export const permissionResources = workspaceProcedure.query(async ({ ctx }) => {
  const workspaceId = ctx.workspace.id;

  const [
    projectRows,
    appRows,
    environmentRows,
    deploymentRows,
    namespaceRows,
    overrideRows,
    roleRows,
    permissionRows,
    identityRows,
  ] = await Promise.all([
    db
      .select({ id: projects.id, name: projects.slug })
      .from(projects)
      .where(eq(projects.workspaceId, workspaceId))
      .orderBy(desc(projects.updatedAt))
      .limit(limit),
    db
      .select({ id: apps.id, name: apps.slug, projectId: apps.projectId })
      .from(apps)
      .where(eq(apps.workspaceId, workspaceId))
      .orderBy(desc(apps.updatedAt))
      .limit(limit),
    db
      .select({
        id: environments.id,
        name: environments.slug,
        projectId: environments.projectId,
        appId: environments.appId,
      })
      .from(environments)
      .where(eq(environments.workspaceId, workspaceId))
      .orderBy(desc(environments.updatedAt))
      .limit(limit),
    db
      .select({
        id: deployments.id,
        name: deployments.gitCommitMessage,
        projectId: deployments.projectId,
        appId: deployments.appId,
        environmentId: deployments.environmentId,
      })
      .from(deployments)
      .where(eq(deployments.workspaceId, workspaceId))
      .orderBy(desc(deployments.createdAt), desc(deployments.id))
      .limit(limit),
    db
      .select({ id: ratelimitNamespaces.id, name: ratelimitNamespaces.name })
      .from(ratelimitNamespaces)
      .where(eq(ratelimitNamespaces.workspaceId, workspaceId))
      .orderBy(desc(ratelimitNamespaces.updatedAtM))
      .limit(limit),
    db
      .select({
        id: ratelimitOverrides.id,
        name: ratelimitOverrides.identifier,
        namespaceId: ratelimitOverrides.namespaceId,
      })
      .from(ratelimitOverrides)
      .where(eq(ratelimitOverrides.workspaceId, workspaceId))
      .orderBy(desc(ratelimitOverrides.updatedAtM))
      .limit(limit),
    db
      .select({ id: roles.id, name: roles.name })
      .from(roles)
      .where(eq(roles.workspaceId, workspaceId))
      .orderBy(desc(roles.updatedAtM))
      .limit(limit),
    db
      .select({ id: permissions.id, name: permissions.slug })
      .from(permissions)
      .where(eq(permissions.workspaceId, workspaceId))
      .orderBy(desc(permissions.updatedAtM))
      .limit(limit),
    db
      .select({ id: identities.id, name: identities.externalId })
      .from(identities)
      .where(eq(identities.workspaceId, workspaceId))
      .orderBy(desc(identities.updatedAt))
      .limit(limit),
  ]);

  return {
    projects: projectRows,
    apps: appRows,
    environments: environmentRows,
    deployments: deploymentRows.map((deployment) => ({
      ...deployment,
      name: deployment.name ?? deployment.id,
    })),
    ratelimitNamespaces: namespaceRows,
    ratelimitOverrides: overrideRows,
    roles: roleRows,
    permissions: permissionRows,
    identities: identityRows,
  };
});

import { and, asc, db, eq, isNull, schema } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { workspaceProcedure } from "../../../trpc";
import { isDefaultProject } from "../project/get-default";

type Row = Awaited<ReturnType<typeof selectRows>>[number];

type EnvVarVariable = {
  id: string;
  key: string;
  type: Row["type"];
  environments: {
    id: string;
    slug: string;
    kind: Row["environmentKind"];
    changedAt: number;
  }[];
};

type EnvVarAppGroup = {
  project: { id: string; name: string; isDefault: boolean };
  app: {
    id: string;
    name: string;
    sourceType: Row["appSourceType"];
    repositoryFullName: string | null;
  };
  variables: EnvVarVariable[];
};

export const listEnvVars = workspaceProcedure.query(async ({ ctx }): Promise<EnvVarAppGroup[]> => {
  try {
    return groupByApp(await selectRows(ctx.workspace.id));
  } catch (error) {
    console.error("Failed to list environment variables", error);
    throw new TRPCError({
      code: "INTERNAL_SERVER_ERROR",
      message: "Failed to list environment variables",
    });
  }
});

function selectRows(workspaceId: string) {
  const {
    apps,
    projects,
    environments,
    appEnvironmentVariables: vars,
    githubRepoConnections: repos,
  } = schema;
  return db
    .select({
      id: vars.id,
      key: vars.key,
      type: vars.type,
      createdAt: vars.createdAt,
      updatedAt: vars.updatedAt,
      projectId: projects.id,
      projectName: projects.name,
      projectIsDefault: isDefaultProject,
      appId: apps.id,
      appName: apps.name,
      appSourceType: apps.sourceType,
      repositoryFullName: repos.repositoryFullName,
      environmentId: environments.id,
      environmentSlug: environments.slug,
      environmentKind: environments.kind,
    })
    .from(apps)
    .innerJoin(projects, and(eq(projects.id, apps.projectId), isNull(projects.deletedAt)))
    .innerJoin(environments, and(eq(environments.appId, apps.id), isNull(environments.deletedAt)))
    .leftJoin(repos, eq(repos.appId, apps.id))
    .innerJoin(vars, and(eq(vars.appId, apps.id), eq(vars.environmentId, environments.id)))
    .where(and(eq(apps.workspaceId, workspaceId), isNull(apps.deletedAt)))
    .orderBy(
      asc(projects.name),
      asc(apps.name),
      asc(apps.id),
      asc(vars.key),
      asc(vars.type),
      // MySQL sorts an ENUM by declaration order, so production comes first.
      asc(environments.kind),
      asc(environments.slug),
    );
}

function groupByApp(rows: Row[]): EnvVarAppGroup[] {
  const groups: EnvVarAppGroup[] = [];
  let group: EnvVarAppGroup | undefined;
  let variable: EnvVarVariable | undefined;

  for (const row of rows) {
    if (group?.app.id !== row.appId) {
      group = {
        project: { id: row.projectId, name: row.projectName, isDefault: row.projectIsDefault },
        app: {
          id: row.appId,
          name: row.appName,
          sourceType: row.appSourceType,
          repositoryFullName: row.repositoryFullName,
        },
        variables: [],
      };
      variable = undefined;
      groups.push(group);
    }
    if (variable?.key !== row.key || variable.type !== row.type) {
      variable = { id: row.id, key: row.key, type: row.type, environments: [] };
      group.variables.push(variable);
    }
    variable.environments.push({
      id: row.environmentId,
      slug: row.environmentSlug,
      kind: row.environmentKind,
      changedAt: row.updatedAt ?? row.createdAt,
    });
  }

  return groups;
}

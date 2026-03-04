import { and, db, eq, inArray } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { appEnvironmentVariables, apps, environments } from "@unkey/db/src/schema";
import { z } from "zod";
import { workspaceProcedure } from "../../../trpc";

const envVarOutputSchema = z.object({
  id: z.string(),
  key: z.string(),
  value: z.string(),
  type: z.enum(["recoverable", "writeonly"]),
  description: z.string().nullable(),
});

const environmentOutputSchema = z.object({
  id: z.string(),
  variables: z.array(envVarOutputSchema),
});

export const listEnvVars = workspaceProcedure
  .input(
    z.object({
      projectId: z.string(),
    }),
  )
  .query(async ({ ctx, input }) => {
    try {
      // Fetch all environments for this project (needed for slugs)
      const envs = await db.query.environments.findMany({
        where: and(
          eq(environments.workspaceId, ctx.workspace.id),
          eq(environments.projectId, input.projectId),
        ),
        columns: {
          id: true,
          slug: true,
        },
      });

      if (envs.length === 0) {
        return {};
      }

      // Fetch all apps for this project to get their IDs
      const projectApps = await db.query.apps.findMany({
        where: and(
          eq(apps.workspaceId, ctx.workspace.id),
          eq(apps.projectId, input.projectId),
        ),
        columns: {
          id: true,
          environmentId: true,
        },
      });

      const appIds = projectApps.map((a) => a.id);

      if (appIds.length === 0) {
        // No apps yet — return empty environments
        const result: Record<string, z.infer<typeof environmentOutputSchema>> = {};
        for (const env of envs) {
          result[env.slug] = { id: env.id, variables: [] };
        }
        return result;
      }

      // Build a lookup from environmentId -> appIds for that environment
      const envIdToAppIds = new Map<string, string[]>();
      for (const app of projectApps) {
        const existing = envIdToAppIds.get(app.environmentId) ?? [];
        existing.push(app.id);
        envIdToAppIds.set(app.environmentId, existing);
      }

      // Fetch all app environment variables in one query
      const allVariables = await db.query.appEnvironmentVariables.findMany({
        where: and(
          eq(appEnvironmentVariables.workspaceId, ctx.workspace.id),
          inArray(appEnvironmentVariables.appId, appIds),
        ),
        columns: {
          id: true,
          appId: true,
          key: true,
          value: true,
          type: true,
          description: true,
        },
      });

      // Build a lookup from appId -> environmentId
      const appIdToEnvId = new Map<string, string>();
      for (const app of projectApps) {
        appIdToEnvId.set(app.id, app.environmentId);
      }

      const result: Record<string, z.infer<typeof environmentOutputSchema>> = {};

      for (const env of envs) {
        const vars = allVariables.filter((v) => appIdToEnvId.get(v.appId) === env.id);

        result[env.slug] = {
          id: env.id,
          variables: vars.map((v) => ({
            id: v.id,
            key: v.key,
            // Decrypted by decrypt endpoint
            value: "••••••••",
            type: v.type,
            description: v.description,
          })),
        };
      }

      return result;
    } catch {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to fetch environment variables",
      });
    }
  });

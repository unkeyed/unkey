import { and, db, eq, inArray } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { appEnvironmentVariables, environments } from "@unkey/db/src/schema";
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
      // Fetch all environments for this project (needed for slugs and appId)
      const envs = await db.query.environments.findMany({
        where: and(
          eq(environments.workspaceId, ctx.workspace.id),
          eq(environments.projectId, input.projectId),
        ),
        columns: {
          id: true,
          slug: true,
          appId: true,
        },
      });

      if (envs.length === 0) {
        return {};
      }

      // Collect unique app IDs from environments
      const appIds = [...new Set(envs.map((e) => e.appId))];

      if (appIds.length === 0) {
        const result: Record<string, z.infer<typeof environmentOutputSchema>> = {};
        for (const env of envs) {
          result[env.slug] = { id: env.id, variables: [] };
        }
        return result;
      }

      // Fetch all app environment variables in one query
      const allVariables = await db.query.appEnvironmentVariables.findMany({
        where: and(
          eq(appEnvironmentVariables.workspaceId, ctx.workspace.id),
          inArray(appEnvironmentVariables.appId, appIds),
        ),
        columns: {
          id: true,
          environmentId: true,
          key: true,
          value: true,
          type: true,
          description: true,
        },
      });

      const result: Record<string, z.infer<typeof environmentOutputSchema>> = {};

      for (const env of envs) {
        const vars = allVariables.filter((v) => v.environmentId === env.id);

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

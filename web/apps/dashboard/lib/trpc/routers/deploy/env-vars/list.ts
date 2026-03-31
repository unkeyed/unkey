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
  updatedAt: z.number(),
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

      const envIds = envs.map((e) => e.id);

      const allVariables = await db.query.appEnvironmentVariables.findMany({
        where: and(
          eq(appEnvironmentVariables.workspaceId, ctx.workspace.id),
          inArray(appEnvironmentVariables.environmentId, envIds),
        ),
        columns: {
          id: true,
          environmentId: true,
          key: true,
          type: true,
          description: true,
          createdAt: true,
          updatedAt: true,
        },
      });

      const varsByEnvId = new Map<string, z.infer<typeof envVarOutputSchema>[]>();
      for (const v of allVariables) {
        const mapped = {
          id: v.id,
          key: v.key,
          value: "••••••••",
          type: v.type,
          description: v.description,
          updatedAt: v.updatedAt ?? v.createdAt,
        };
        const existing = varsByEnvId.get(v.environmentId);
        if (existing) {
          existing.push(mapped);
        } else {
          varsByEnvId.set(v.environmentId, [mapped]);
        }
      }

      const result: Record<string, z.infer<typeof environmentOutputSchema>> = {};
      for (const env of envs) {
        result[env.slug] = {
          id: env.id,
          variables: varsByEnvId.get(env.id) ?? [],
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

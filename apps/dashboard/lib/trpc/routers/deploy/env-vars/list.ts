import { and, db, eq, inArray } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { environmentVariables, environments } from "@unkey/db/src/schema";
import { z } from "zod";
import { requireUser, requireWorkspace, t } from "../../../trpc";

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

export const listEnvVars = t.procedure
  .use(requireUser)
  .use(requireWorkspace)
  .input(
    z.object({
      projectId: z.string(),
    }),
  )
  .query(async ({ ctx, input }) => {
    try {
      // Fetch all environments for this project
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

      const envIds = envs.map((e) => e.id);

      if (envIds.length === 0) {
        return {};
      }

      // Fetch all environment variables in one query
      const allVariables = await db.query.environmentVariables.findMany({
        where: and(
          eq(environmentVariables.workspaceId, ctx.workspace.id),
          inArray(environmentVariables.environmentId, envIds),
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

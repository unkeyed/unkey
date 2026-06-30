import { and, db, eq, inArray, notInArray, schema } from "@/lib/db";
import { envVarKeySchema } from "@/lib/schemas/env-var";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { workspaceProcedure } from "../../../trpc";

// Renames a set of environment variables to a new key in a single operation.
// This exists so variables that share a key across environments (e.g. prod and
// dev) can be renamed together and stay in sync. Only the key is touched, so
// values are never re-encrypted and writeonly variables can be renamed safely.
export const renameEnvVars = workspaceProcedure
  .input(
    z.object({
      envVarIds: z.array(z.string()).min(1),
      key: envVarKeySchema,
    }),
  )
  .mutation(async ({ ctx, input }) => {
    try {
      const envVars = await db.query.appEnvironmentVariables.findMany({
        where: and(
          inArray(schema.appEnvironmentVariables.id, input.envVarIds),
          eq(schema.appEnvironmentVariables.workspaceId, ctx.workspace.id),
        ),
        columns: {
          id: true,
          appId: true,
          environmentId: true,
          key: true,
        },
      });

      if (envVars.length === 0) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "Environment variable(s) not found",
        });
      }

      // All targeted variables must belong to the same app, otherwise the
      // unique (appId, environmentId, key) constraint cannot be reasoned about.
      const appIds = new Set(envVars.map((v) => v.appId));
      if (appIds.size > 1) {
        throw new TRPCError({
          code: "BAD_REQUEST",
          message: "Cannot rename environment variables across different apps",
        });
      }
      const appId = envVars[0].appId;

      // Two targets in the same environment would both resolve to the same new
      // key and violate the unique (appId, environmentId, key) constraint. Reject
      // that here so it surfaces as an actionable error rather than a generic 500.
      const environmentIds = [...new Set(envVars.map((v) => v.environmentId))];
      if (environmentIds.length !== envVars.length) {
        throw new TRPCError({
          code: "BAD_REQUEST",
          message: "Cannot rename multiple variables in the same environment to the same key",
        });
      }

      // A rename would collide if another variable (not in this set) already
      // uses the new key in any of the targeted environments.
      const conflicts = await db.query.appEnvironmentVariables.findMany({
        where: and(
          eq(schema.appEnvironmentVariables.workspaceId, ctx.workspace.id),
          eq(schema.appEnvironmentVariables.appId, appId),
          eq(schema.appEnvironmentVariables.key, input.key),
          inArray(schema.appEnvironmentVariables.environmentId, environmentIds),
          notInArray(schema.appEnvironmentVariables.id, input.envVarIds),
        ),
        columns: { id: true },
      });

      if (conflicts.length > 0) {
        throw new TRPCError({
          code: "CONFLICT",
          message: `A variable named "${input.key}" already exists in one of these environments`,
        });
      }

      const result = await db
        .update(schema.appEnvironmentVariables)
        .set({ key: input.key })
        .where(
          and(
            inArray(schema.appEnvironmentVariables.id, input.envVarIds),
            eq(schema.appEnvironmentVariables.workspaceId, ctx.workspace.id),
          ),
        );

      return { updated: result[0].affectedRows };
    } catch (error) {
      if (error instanceof TRPCError) {
        throw error;
      }

      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to rename environment variable(s)",
      });
    }
  });

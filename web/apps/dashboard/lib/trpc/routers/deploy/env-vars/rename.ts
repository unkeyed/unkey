import { insertAuditLogs } from "@/lib/audit";
import { and, db, eq, inArray, schema } from "@/lib/db";
import { envVarKeySchema } from "@/lib/schemas/env-var";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { ratelimit, withRatelimit, workspaceProcedure } from "../../../trpc";

// Renames a variable across environments in one statement.
//
// The v2 API identifies a variable by key, so a rename there is a write of the
// new key and a delete of the old one. That write needs the value, which the
// API never returns for a writeonly variable. It also cannot refuse a rename
// onto a key that is in use. This changes only the key column.
export const renameEnvVars = workspaceProcedure
  .use(withRatelimit(ratelimit.update))
  .input(
    z.object({
      appId: z.string().min(1),
      environmentIds: z.array(z.string()).min(1),
      key: z.string().min(1),
      newKey: envVarKeySchema,
    }),
  )
  .mutation(async ({ ctx, input }) => {
    if (input.key === input.newKey) {
      return { updated: 0 };
    }

    try {
      const environmentIds = [...new Set(input.environmentIds)];

      const targets = await db.query.appEnvironmentVariables.findMany({
        where: and(
          eq(schema.appEnvironmentVariables.workspaceId, ctx.workspace.id),
          eq(schema.appEnvironmentVariables.appId, input.appId),
          eq(schema.appEnvironmentVariables.key, input.key),
          inArray(schema.appEnvironmentVariables.environmentId, environmentIds),
        ),
        columns: { id: true },
        with: { environment: { columns: { id: true, slug: true } } },
      });

      if (targets.length === 0) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "Environment variable(s) not found",
        });
      }

      // Another variable can already hold the new key in one of these
      // environments. Fail the rename instead of replacing it.
      const conflicts = await db.query.appEnvironmentVariables.findMany({
        where: and(
          eq(schema.appEnvironmentVariables.workspaceId, ctx.workspace.id),
          eq(schema.appEnvironmentVariables.appId, input.appId),
          eq(schema.appEnvironmentVariables.key, input.newKey),
          inArray(schema.appEnvironmentVariables.environmentId, environmentIds),
        ),
        columns: { id: true },
      });

      if (conflicts.length > 0) {
        throw new TRPCError({
          code: "CONFLICT",
          message: `A variable named "${input.newKey}" already exists in one of these environments`,
        });
      }

      return await db.transaction(async (tx) => {
        const result = await tx
          .update(schema.appEnvironmentVariables)
          .set({ key: input.newKey })
          .where(
            and(
              inArray(
                schema.appEnvironmentVariables.id,
                targets.map((t) => t.id),
              ),
              eq(schema.appEnvironmentVariables.workspaceId, ctx.workspace.id),
            ),
          );

        await insertAuditLogs(
          tx,
          targets.map(({ environment }) => ({
            workspaceId: ctx.workspace.id,
            actor: { type: "user", id: ctx.user.id },
            event: "environment.update",
            description: `Renamed environment variable ${input.key} to ${input.newKey} for environment ${environment.id}`,
            resources: [
              {
                type: "environment",
                id: environment.id,
                name: environment.slug,
                meta: { key: input.newKey, previousKey: input.key },
              },
            ],
            context: { location: ctx.audit.location, userAgent: ctx.audit.userAgent },
          })),
        );

        return { updated: result[0].affectedRows };
      });
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

import { insertAuditLogs } from "@/lib/audit";
import { and, db, eq, inArray, or, schema } from "@/lib/db";
import { plural } from "@/lib/fmt";
import { z } from "zod";
import { ratelimit, withRatelimit, workspaceProcedure } from "../../../trpc";
import { envVarAuditLogs, toEnvVarWriteError } from "./env-var-write";

// Converts recoverable variables to writeonly. The change is one way.
//
// The v2 API cannot change the kind on its own, because its input needs a
// value. The dashboard would have to read each secret and send the plaintext
// again only to relabel it. This changes the type column and keeps the
// ciphertext.
export const makeSensitive = workspaceProcedure
  .use(withRatelimit(ratelimit.update))
  .input(
    z.object({
      appId: z.string().min(1),
      targets: z
        .array(
          z.object({
            environmentId: z.string().min(1),
            key: z.string().min(1),
          }),
        )
        .min(1),
    }),
  )
  .mutation(async ({ ctx, input }) => {
    try {
      return await db.transaction(async (tx) => {
        const targets = await tx.query.appEnvironmentVariables.findMany({
          where: and(
            eq(schema.appEnvironmentVariables.workspaceId, ctx.workspace.id),
            eq(schema.appEnvironmentVariables.appId, input.appId),
            eq(schema.appEnvironmentVariables.type, "recoverable"),
            or(
              ...input.targets.map((t) =>
                and(
                  eq(schema.appEnvironmentVariables.environmentId, t.environmentId),
                  eq(schema.appEnvironmentVariables.key, t.key),
                ),
              ),
            ),
          ),
          columns: { id: true, key: true },
          with: { environment: { columns: { id: true, slug: true } } },
        });
        if (targets.length === 0) {
          return { updated: 0 };
        }

        const result = await tx
          .update(schema.appEnvironmentVariables)
          .set({ type: "writeonly" })
          .where(
            and(
              eq(schema.appEnvironmentVariables.workspaceId, ctx.workspace.id),
              eq(schema.appEnvironmentVariables.type, "recoverable"),
              inArray(
                schema.appEnvironmentVariables.id,
                targets.map((t) => t.id),
              ),
            ),
          );

        await insertAuditLogs(
          tx,
          envVarAuditLogs({
            workspaceId: ctx.workspace.id,
            userId: ctx.user.id,
            context: { location: ctx.audit.location, userAgent: ctx.audit.userAgent },
            changes: targets.map(({ environment, key }) => ({
              action: "makeSensitive" as const,
              environment,
              key,
            })),
          }),
        );

        return { updated: result[0].affectedRows };
      });
    } catch (error) {
      throw toEnvVarWriteError(
        error,
        `Failed to mark ${plural(input.targets.length, "environment variable")} as sensitive`,
      );
    }
  });

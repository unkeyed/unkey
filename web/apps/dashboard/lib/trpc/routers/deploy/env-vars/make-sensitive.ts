import { and, db, eq, or, schema } from "@/lib/db";
import { envVarKeySchema } from "@/lib/schemas/env-var";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { workspaceProcedure } from "../../../trpc";

// Converts recoverable variables to writeonly. The change is one way.
//
// The v2 API cannot change the kind on its own, because its input needs a
// value. The dashboard would have to read each secret and send the plaintext
// again only to relabel it. This changes the type column and keeps the
// ciphertext.
export const makeSensitive = workspaceProcedure
  .input(
    z.object({
      appId: z.string().min(1),
      targets: z
        .array(
          z.object({
            environmentId: z.string().min(1),
            key: envVarKeySchema,
          }),
        )
        .min(1),
    }),
  )
  .mutation(async ({ ctx, input }) => {
    try {
      const result = await db
        .update(schema.appEnvironmentVariables)
        .set({ type: "writeonly" })
        .where(
          and(
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
        );

      return { updated: result[0].affectedRows };
    } catch (error) {
      if (error instanceof TRPCError) {
        throw error;
      }

      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to mark environment variable(s) as sensitive",
      });
    }
  });

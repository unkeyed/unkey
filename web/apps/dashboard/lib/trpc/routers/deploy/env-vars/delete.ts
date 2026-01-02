import { and, db, eq, schema } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { workspaceProcedure } from "../../../trpc";

export const deleteEnvVar = workspaceProcedure
  .input(
    z.object({
      envVarId: z.string(),
    }),
  )
  .mutation(async ({ ctx, input }) => {
    try {
      const result = await db
        .delete(schema.environmentVariables)
        .where(
          and(
            eq(schema.environmentVariables.id, input.envVarId),
            eq(schema.environmentVariables.workspaceId, ctx.workspace.id),
          ),
        );

      if (result.rowsAffected === 0) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "Environment variable not found",
        });
      }
    } catch (error) {
      if (error instanceof TRPCError) {
        throw error;
      }

      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to delete environment variable",
      });
    }
  });

import { and, db, eq, inArray, schema } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { workspaceProcedure } from "../../../trpc";

export const deleteEnvVar = workspaceProcedure
  .input(
    z.object({
      envVarIds: z.array(z.string()).min(1),
    }),
  )
  .mutation(async ({ ctx, input }) => {
    try {
      const result = await db
        .delete(schema.appEnvironmentVariables)
        .where(
          and(
            inArray(schema.appEnvironmentVariables.id, input.envVarIds),
            eq(schema.appEnvironmentVariables.workspaceId, ctx.workspace.id),
          ),
        );

      if (result.rowsAffected === 0) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "Environment variable(s) not found",
        });
      }

      return { deletedCount: result.rowsAffected };
    } catch (error) {
      if (error instanceof TRPCError) {
        throw error;
      }

      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to delete environment variable(s)",
      });
    }
  });

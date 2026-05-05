import { and, db, eq, inArray, schema } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { workspaceProcedure } from "../../../trpc";

export const makeSensitive = workspaceProcedure
  .input(
    z.object({
      envVarIds: z.array(z.string()).min(1),
    }),
  )
  .mutation(async ({ ctx, input }) => {
    try {
      await db
        .update(schema.appEnvironmentVariables)
        .set({ type: "writeonly" })
        .where(
          and(
            inArray(schema.appEnvironmentVariables.id, input.envVarIds),
            eq(schema.appEnvironmentVariables.workspaceId, ctx.workspace.id),
            eq(schema.appEnvironmentVariables.type, "recoverable"),
          ),
        );
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

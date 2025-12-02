import { and, db, eq, schema } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { requireUser, requireWorkspace, t } from "../../../trpc";

export const deleteEnvVar = t.procedure
  .use(requireUser)
  .use(requireWorkspace)
  .input(
    z.object({
      envVarId: z.string(),
    }),
  )
  .mutation(async ({ ctx, input }) => {
    // Verify the env var belongs to this workspace
    const envVar = await db.query.environmentVariables.findFirst({
      where: and(
        eq(schema.environmentVariables.id, input.envVarId),
        eq(schema.environmentVariables.workspaceId, ctx.workspace.id),
      ),
      columns: {
        id: true,
      },
    });

    if (!envVar) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "Environment variable not found",
      });
    }

    await db
      .delete(schema.environmentVariables)
      .where(eq(schema.environmentVariables.id, input.envVarId));

    return { success: true };
  });

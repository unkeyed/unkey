import { TRPCError } from "@trpc/server";
import { workspaceProcedure } from "../../trpc";

export const getWorkspaceById = workspaceProcedure.query(async ({ ctx }) => {
  const slug = ctx.workspace.slug;

  if (!slug) {
    throw new TRPCError({
      code: "NOT_FOUND",
      message: "Workspace not found",
    });
  }

  return { slug };
});

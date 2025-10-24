import { db } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { requireUser, t } from "../../trpc";

export const getWorkspaceById = t.procedure
  .use(requireUser)
  .input(
    z.object({
      workspaceId: z.string().min(1, "Workspace ID is required"),
    }),
  )
  .query(async ({ input }) => {
    try {
      const workspace = await db.query.workspaces.findFirst({
        where: (table, { eq, and, isNull }) =>
          and(eq(table.id, input.workspaceId), isNull(table.deletedAtM)),
        with: {
          quotas: true,
        },
      });

      if (!workspace) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "Workspace not found",
        });
      }

      return workspace;
    } catch (error) {
      // If it's already a TRPCError, re-throw it
      if (error instanceof TRPCError) {
        throw error;
      }

      console.error("Error fetching workspace by ID:", error);
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to fetch workspace data",
        cause: error,
      });
    }
  });

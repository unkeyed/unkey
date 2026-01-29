import { db } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { z } from "zod";

export const listCustomDomains = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .input(z.object({ projectId: z.string() }))
  .query(async ({ input, ctx }) => {
    // Verify project belongs to workspace
    const project = await db.query.projects.findFirst({
      where: (table, { eq, and }) =>
        and(eq(table.id, input.projectId), eq(table.workspaceId, ctx.workspace.id)),
      columns: {
        id: true,
      },
    });

    if (!project) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "Project not found",
      });
    }

    try {
      const domains = await db.query.customDomains.findMany({
        where: (table, { eq }) => eq(table.projectId, input.projectId),
        columns: {
          id: true,
          domain: true,
          workspaceId: true,
          projectId: true,
          environmentId: true,
          verificationStatus: true,
          targetCname: true,
          checkAttempts: true,
          lastCheckedAt: true,
          verificationError: true,
          createdAt: true,
          updatedAt: true,
        },
        orderBy: (table, { desc }) => desc(table.createdAt),
      });

      return domains.map((d) => ({
        id: d.id,
        domain: d.domain,
        workspaceId: d.workspaceId,
        projectId: d.projectId,
        environmentId: d.environmentId,
        verificationStatus: d.verificationStatus,
        targetCname: d.targetCname,
        checkAttempts: d.checkAttempts,
        lastCheckedAt: d.lastCheckedAt,
        verificationError: d.verificationError,
        createdAt: d.createdAt,
        updatedAt: d.updatedAt,
      }));
    } catch (error) {
      console.error("List custom domains failed:", error);
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to list custom domains",
      });
    }
  });

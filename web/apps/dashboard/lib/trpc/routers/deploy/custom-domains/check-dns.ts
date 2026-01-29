import { db } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { z } from "zod";

export const checkDns = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .input(
    z.object({
      domain: z.string().min(1, "Domain is required"),
      projectId: z.string().min(1, "Project ID is required"),
    }),
  )
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

    // Get the domain record
    const domainRecord = await db.query.customDomains.findFirst({
      where: (table, { eq, and }) =>
        and(eq(table.domain, input.domain), eq(table.projectId, input.projectId)),
      columns: {
        id: true,
        domain: true,
        verificationToken: true,
        ownershipVerified: true,
        cnameVerified: true,
        targetCname: true,
        verificationStatus: true,
      },
    });

    if (!domainRecord) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "Domain not found",
      });
    }

    // Return the current verification state from the database
    // The actual DNS checks happen in the backend worker
    return {
      domain: domainRecord.domain,
      verificationToken: domainRecord.verificationToken,
      ownershipVerified: domainRecord.ownershipVerified,
      cnameVerified: domainRecord.cnameVerified,
      targetCname: domainRecord.targetCname,
      verificationStatus: domainRecord.verificationStatus,
    };
  });

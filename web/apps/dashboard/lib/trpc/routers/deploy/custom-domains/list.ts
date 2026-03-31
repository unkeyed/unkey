import { createHmac } from "node:crypto";
import { db } from "@/lib/db";
import { env } from "@/lib/env";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { z } from "zod";

/**
 * Computes a deterministic verification token for a domain using HMAC-SHA256.
 * Must match the Go implementation in svc/ctrl/worker/customdomain/service.go
 * and svc/ctrl/services/customdomain/service.go.
 */
function verificationToken(domainId: string, signingKey: string): string {
  return createHmac("sha256", signingKey).update(domainId).digest("hex").slice(0, 32);
}

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
      const signingKey = env().DOMAIN_SIGNING_KEY ?? "";

      const domains = await db.query.customDomains.findMany({
        where: (table, { eq }) => eq(table.projectId, input.projectId),
        columns: {
          id: true,
          domain: true,
          workspaceId: true,
          projectId: true,
          appId: true,
          environmentId: true,
          verificationStatus: true,
          ownershipVerified: true,
          cnameVerified: true,
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
        appId: d.appId,
        environmentId: d.environmentId,
        verificationStatus: d.verificationStatus,
        verificationToken: verificationToken(d.id, signingKey),
        ownershipVerified: d.ownershipVerified,
        cnameVerified: d.cnameVerified,
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

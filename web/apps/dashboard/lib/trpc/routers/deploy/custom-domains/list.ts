import { CustomDomainService, CustomDomainStatus } from "@/gen/proto/ctrl/v1/custom_domain_pb";
import { db } from "@/lib/db";
import { env } from "@/lib/env";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { createClient } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import { TRPCError } from "@trpc/server";
import { z } from "zod";

// Convert proto enum to string status
function statusToString(status: CustomDomainStatus): string {
  switch (status) {
    case CustomDomainStatus.PENDING:
      return "pending";
    case CustomDomainStatus.VERIFYING:
      return "verifying";
    case CustomDomainStatus.VERIFIED:
      return "verified";
    case CustomDomainStatus.FAILED:
      return "failed";
    default:
      return "pending";
  }
}

export const listCustomDomains = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .input(z.object({ projectId: z.string() }))
  .query(async ({ input, ctx }) => {
    const { CTRL_URL, CTRL_API_KEY } = env();
    if (!CTRL_URL || !CTRL_API_KEY) {
      throw new TRPCError({
        code: "PRECONDITION_FAILED",
        message: "ctrl service is not configured",
      });
    }

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

    const ctrl = createClient(
      CustomDomainService,
      createConnectTransport({
        baseUrl: CTRL_URL,
        interceptors: [
          (next) => (req) => {
            req.header.set("Authorization", `Bearer ${CTRL_API_KEY}`);
            return next(req);
          },
        ],
      }),
    );

    try {
      const response = await ctrl.listCustomDomains({
        projectId: input.projectId,
      });

      return response.domains.map((d) => ({
        id: d.id,
        domain: d.domain,
        workspaceId: d.workspaceId,
        projectId: d.projectId,
        environmentId: d.environmentId,
        verificationStatus: statusToString(d.verificationStatus),
        targetCname: d.targetCname,
        checkAttempts: d.checkAttempts,
        lastCheckedAt: d.lastCheckedAt ? Number(d.lastCheckedAt) : null,
        verificationError: d.verificationError || null,
        createdAt: Number(d.createdAt),
        updatedAt: d.updatedAt ? Number(d.updatedAt) : null,
      }));
    } catch (error) {
      console.error("List custom domains failed:", error);
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to list custom domains",
      });
    }
  });

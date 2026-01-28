import { CustomDomainService } from "@/gen/proto/ctrl/v1/custom_domain_pb";
import { db } from "@/lib/db";
import { env } from "@/lib/env";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { createClient } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import { TRPCError } from "@trpc/server";
import { z } from "zod";

export const retryVerification = workspaceProcedure
  .use(withRatelimit(ratelimit.create))
  .input(
    z.object({
      domain: z.string().min(1, "Domain is required"),
      projectId: z.string().min(1, "Project ID is required"),
    }),
  )
  .mutation(async ({ input, ctx }) => {
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
      const response = await ctrl.retryVerification({
        domain: input.domain,
      });

      return { status: response.status };
    } catch (error) {
      console.error("Retry verification failed:", error);

      if (error instanceof Error && error.message.includes("not found")) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "Domain not found",
        });
      }

      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to retry verification",
      });
    }
  });

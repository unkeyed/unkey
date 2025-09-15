import { insertAuditLogs } from "@/lib/audit";
import { db } from "@/lib/db";
import { env } from "@/lib/env";
import { ratelimit, requireUser, requireWorkspace, t, withRatelimit } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { z } from "zod";

export const rollback = t.procedure
  .use(requireUser)
  .use(requireWorkspace)
  .use(withRatelimit(ratelimit.update))
  .input(
    z.object({
      hostname: z.string().min(1, "Hostname is required"),
      targetVersionId: z.string().min(1, "Target version ID is required"),
    }),
  )
  .mutation(async ({ input, ctx }) => {
    const { hostname, targetVersionId } = input;
    const workspaceId = ctx.workspace.id;

    // Validate that ctrl service URL is configured
    const ctrlUrl = env().CTRL_URL;
    if (!ctrlUrl) {
      throw new Error("ctrl service is not configured");
    }

    try {
      // Verify the target deployment exists and belongs to this workspace
      const deployment = await db.query.deployments.findFirst({
        where: (table, { eq, and }) =>
          and(eq(table.id, targetVersionId), eq(table.workspaceId, workspaceId)),
        columns: {
          id: true,
          status: true,
        },
        with: {
          project: {
            columns: {
              name: true,
            },
          },
        },
      });

      if (!deployment) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "Deployment not found or access denied",
        });
      }

      if (deployment.status !== "ready") {
        throw new TRPCError({
          code: "PRECONDITION_FAILED",
          message: `Deployment ${targetVersionId} is not ready (status: ${deployment.status})`,
        });
      }

      // Make request to ctrl service rollback endpoint
      const rollbackRequest = {
        hostname,
        target_version_id: targetVersionId,
        workspace_id: workspaceId,
      };

      const response = await fetch(`${ctrlUrl}/ctrl.v1.RoutingService/Rollback`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify(rollbackRequest),
      });

      if (!response.ok) {
        const errorText = await response.text();
        let errorMessage = "Failed to initiate rollback";

        try {
          const errorData = JSON.parse(errorText);
          if (errorData.message) {
            errorMessage = errorData.message;
          }
        } catch {
          // Keep default error message if JSON parsing fails
        }

        // Map common ctrl service errors to appropriate tRPC errors
        if (response.status === 404) {
          throw new TRPCError({
            code: "NOT_FOUND",
            message: errorMessage,
          });
        }
        if (response.status === 412) {
          throw new TRPCError({
            code: "PRECONDITION_FAILED",
            message: errorMessage,
          });
        }
        if (response.status === 401 || response.status === 403) {
          throw new TRPCError({
            code: "FORBIDDEN",
            message: "Unauthorized to perform rollback",
          });
        }

        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message: errorMessage,
        });
      }

      const rollbackResponse = await response.json();

      // Log the rollback action for audit purposes
      await insertAuditLogs(db, {
        workspaceId: ctx.workspace.id,
        actor: { type: "user", id: ctx.user.id },
        event: "deployment.rollback",
        description: `Rolled back ${hostname} to deployment ${targetVersionId}`,
        resources: [
          {
            type: "deployment",
            id: targetVersionId,
            name: deployment.project?.name || "Unknown",
          },
        ],
        context: {
          location: ctx.audit?.location ?? "unknown",
          userAgent: ctx.audit?.userAgent ?? "unknown",
        },
      });

      return {
        previousVersionId: rollbackResponse.previous_version_id,
        newVersionId: rollbackResponse.new_version_id,
        effectiveAt: rollbackResponse.effective_at,
      };
    } catch (error) {
      if (error instanceof TRPCError) {
        throw error;
      }

      console.error("Rollback request failed:", error);
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to communicate with control service",
      });
    }
  });

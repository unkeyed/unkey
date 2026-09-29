import { type UnkeyAuditLog, insertAuditLogs } from "@/lib/audit";
import { and, eq, schema } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import type { z } from "zod";
import type { target } from "./schemas";

type Target = z.infer<typeof target>;
const duplicateMessage = "This app is already connected, or the name is already in use.";

export async function audit(
  tx: Parameters<typeof insertAuditLogs>[0],
  ctx: {
    workspace: { id: string };
    user: { id: string };
    audit: UnkeyAuditLog["context"];
  },
  appId: string,
  description: string,
) {
  await insertAuditLogs(tx, {
    workspaceId: ctx.workspace.id,
    actor: { type: "user", id: ctx.user.id },
    event: "app.update",
    description,
    resources: [{ type: "app", id: appId }],
    context: ctx.audit,
  });
}
export async function lockPinnedTarget(
  tx: Parameters<typeof insertAuditLogs>[0],
  workspaceId: string,
  projectId: string,
  targetAppId: string,
  input: Target,
) {
  if (input.targetType !== "deployment") {
    return;
  }
  const [deployment] = await tx
    .select({
      status: schema.deployments.status,
      desiredState: schema.deployments.desiredState,
    })
    .from(schema.deployments)
    .where(
      and(
        eq(schema.deployments.id, input.targetDeploymentId),
        eq(schema.deployments.workspaceId, workspaceId),
        eq(schema.deployments.projectId, projectId),
        eq(schema.deployments.appId, targetAppId),
      ),
    )
    .for("update");
  if (deployment?.status !== "ready" || deployment.desiredState !== "running") {
    throw new TRPCError({
      code: "CONFLICT",
      message: "The target deployment is no longer running. Select a ready deployment.",
    });
  }
}

export async function mutateWithDuplicateHandling(fn: () => Promise<unknown>) {
  try {
    await fn();
  } catch (error) {
    if (
      error instanceof Error &&
      (("code" in error && error.code === "ER_DUP_ENTRY") ||
        (error.cause instanceof Error &&
          "code" in error.cause &&
          error.cause.code === "ER_DUP_ENTRY"))
    ) {
      throw new TRPCError({ code: "CONFLICT", message: duplicateMessage });
    }
    throw error;
  }
}

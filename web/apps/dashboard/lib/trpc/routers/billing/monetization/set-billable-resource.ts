import { insertAuditLogs } from "@/lib/audit";
import { and, db, eq, schema } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { newId } from "@unkey/id";
import { z } from "zod";
import { requireWorkspaceAdmin, workspaceProcedure } from "../../../trpc";

export const setBillableResourceInputSchema = z.object({
  resourceType: z.enum(["keyspace", "namespace"]),
  resourceId: z.string().min(1),
  enabled: z.boolean(),
});

/**
 * Confirms a keyspace/namespace id belongs to the workspace. A keyspace id is a
 * key_auth id (referenced by an API); a namespace id is a ratelimit_namespace
 * id. Checked only when enabling so the UI-offered set (listBillableResources)
 * and the write path agree — the billing query is already workspace-scoped, so
 * a foreign id cannot leak data, but persisting one is meaningless config.
 */
async function resourceBelongsToWorkspace(
  workspaceId: string,
  resourceType: "keyspace" | "namespace",
  resourceId: string,
): Promise<boolean> {
  if (resourceType === "keyspace") {
    const api = await db.query.apis.findFirst({
      where: (table, { and, eq, isNull }) =>
        and(
          eq(table.keyAuthId, resourceId),
          eq(table.workspaceId, workspaceId),
          isNull(table.deletedAtM),
        ),
    });
    return Boolean(api);
  }
  const namespace = await db.query.ratelimitNamespaces.findFirst({
    where: (table, { and, eq, isNull }) =>
      and(eq(table.id, resourceId), eq(table.workspaceId, workspaceId), isNull(table.deletedAtM)),
  });
  return Boolean(namespace);
}

/**
 * Enables or disables a keyspace/namespace for end-user billing. Enabling
 * inserts a presence row (idempotent via the unique
 * workspace+type+resource index); disabling removes it. Usage on a resource is
 * billed only while its row exists.
 */
export const setBillableResource = workspaceProcedure
  .use(requireWorkspaceAdmin)
  .input(setBillableResourceInputSchema)
  .mutation(async ({ input, ctx }) => {
    // Enabling a resource that is not in this workspace is a no-op for billing
    // (the usage query is workspace-scoped) but would persist junk config;
    // reject it so the write path matches what the UI can offer. Disabling is
    // left unchecked so a stale row for a since-deleted resource can be removed.
    if (input.enabled) {
      const owned = await resourceBelongsToWorkspace(
        ctx.workspace.id,
        input.resourceType,
        input.resourceId,
      );
      if (!owned) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: `No ${input.resourceType} "${input.resourceId}" in this workspace.`,
        });
      }
    }

    await db.transaction(async (tx) => {
      if (input.enabled) {
        await tx
          .insert(schema.billingBillableResources)
          .values({
            // Opaque unique id; the billing area reuses the rateCard prefix
            // (see set-default.ts) rather than minting a new id namespace.
            id: newId("rateCard"),
            workspaceId: ctx.workspace.id,
            resourceType: input.resourceType,
            resourceId: input.resourceId,
            createdAt: Date.now(),
            updatedAt: null,
          })
          .onDuplicateKeyUpdate({ set: { updatedAt: Date.now() } });
      } else {
        await tx
          .delete(schema.billingBillableResources)
          .where(
            and(
              eq(schema.billingBillableResources.workspaceId, ctx.workspace.id),
              eq(schema.billingBillableResources.resourceType, input.resourceType),
              eq(schema.billingBillableResources.resourceId, input.resourceId),
            ),
          );
      }

      await insertAuditLogs(tx, {
        workspaceId: ctx.workspace.id,
        actor: { type: "user", id: ctx.user.id },
        event: "workspace.update",
        description: `${input.enabled ? "Enabled" : "Disabled"} billing for ${input.resourceType} ${input.resourceId}`,
        resources: [
          {
            type: "workspace",
            id: ctx.workspace.id,
            name: ctx.workspace.name || "Unknown workspace",
            meta: {
              resourceType: input.resourceType,
              resourceId: input.resourceId,
              enabled: input.enabled,
            },
          },
        ],
        context: {
          location: ctx.audit.location,
          userAgent: ctx.audit.userAgent,
        },
      });
    });

    return {
      resourceType: input.resourceType,
      resourceId: input.resourceId,
      enabled: input.enabled,
    };
  });

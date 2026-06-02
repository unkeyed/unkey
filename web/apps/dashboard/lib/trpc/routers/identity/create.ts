import { insertAuditLogs } from "@/lib/audit";
import { type InsertIdentity, db, schema } from "@/lib/db";
import { ratelimitItemSchema } from "@/lib/schemas/ratelimit";
import { TRPCError } from "@trpc/server";
import { newId } from "@unkey/id";
import { z } from "zod";
import { workspaceProcedure } from "../../trpc";

export const createIdentityInputSchema = z.object({
  externalId: z
    .string()
    .transform((s) => s.trim())
    .refine((trimmed) => trimmed.length >= 1, "External ID is required")
    .refine((trimmed) => trimmed.length <= 255, "External ID cannot exceed 255 characters")
    .refine((trimmed) => trimmed !== "", "External ID cannot be only whitespace"),
  meta: z.record(z.string(), z.unknown()).nullable(),
  ratelimits: z.array(ratelimitItemSchema).optional(),
});

export const createIdentity = workspaceProcedure
  .input(createIdentityInputSchema)
  .mutation(async ({ input, ctx }) => {
    const identityId = newId("identity");

    try {
      // We intentionally do NOT pre-check for an existing identity. The unique
      // index `workspace_id_external_id_deleted_idx` is on
      // (workspaceId, externalId, deleted), so a soft-deleted row
      // (deleted=true) never collides with a new active row (deleted=false) —
      // the schema deliberately allows reusing a previously-deleted externalId.
      // A read-then-insert pre-check would also be racy under concurrent
      // creates. Instead we insert directly and translate the unique-constraint
      // violation into a clean CONFLICT below, matching the public Go API
      // handler (svc/api/routes/v2_identities_create_identity/handler.go).

      // Validate that meta is valid if provided
      if (input.meta) {
        try {
          JSON.stringify(input.meta);
        } catch {
          throw new TRPCError({
            code: "BAD_REQUEST",
            message: "The provided metadata is invalid. Please ensure it's valid JSON.",
          });
        }
      }

      await db.transaction(async (tx) => {
        const payload: InsertIdentity = {
          id: identityId,
          externalId: input.externalId,
          workspaceId: ctx.workspace.id,
          createdAt: Date.now(),
          updatedAt: null,
          meta: input.meta,
          environment: "",
          deleted: false,
        };

        await tx.insert(schema.identities).values(payload);

        // Insert ratelimits if provided
        if (input.ratelimits && input.ratelimits.length > 0) {
          const ratelimitValues = input.ratelimits.map((ratelimit) => ({
            id: newId("ratelimit"),
            identityId: identityId,
            duration: ratelimit.refillInterval,
            limit: ratelimit.limit,
            name: ratelimit.name,
            autoApply: ratelimit.autoApply,
            workspaceId: ctx.workspace.id,
            createdAt: Date.now(),
            updatedAt: null,
          }));

          await tx.insert(schema.ratelimits).values(ratelimitValues);
        }

        await insertAuditLogs(tx, {
          workspaceId: ctx.workspace.id,
          actor: { type: "user", id: ctx.user.id },
          event: "identity.create",
          description: `Created identity "${input.externalId}" (${identityId})`,
          resources: [
            {
              type: "identity",
              id: identityId,
              name: input.externalId,
              meta: {
                hasMeta: Boolean(input.meta),
                hasRatelimits: Boolean(input.ratelimits && input.ratelimits.length > 0),
                ratelimitCount: input.ratelimits?.length || 0,
              },
            },
            {
              type: "workspace",
              id: ctx.workspace.id,
              name: ctx.workspace.name || "Unknown workspace",
            },
          ],
          context: {
            location: ctx.audit.location,
            userAgent: ctx.audit.userAgent,
          },
        });
      });

      return {
        identityId,
        externalId: input.externalId,
      };
    } catch (err) {
      if (err instanceof TRPCError) {
        console.error({
          message: "Failed to create identity",
          workspaceId: ctx.workspace.id,
          externalId: input.externalId,
          errorCode: err.code,
          errorMessage: err.message,
        });
        throw err;
      }

      // The unique index covers (workspaceId, externalId, deleted=false), so a
      // duplicate-key error means an *active* identity with this externalId
      // already exists. Soft-deleted rows do not trigger this.
      if (
        err instanceof Error &&
        "code" in err &&
        (err as { code: string }).code === "ER_DUP_ENTRY"
      ) {
        console.info({
          message: "Attempted to create duplicate identity",
          workspaceId: ctx.workspace.id,
          externalId: input.externalId,
        });

        throw new TRPCError({
          code: "CONFLICT",
          message: `An identity with external ID "${input.externalId}" already exists in your workspace.`,
        });
      }

      console.error({
        message: "Failed to create identity",
        workspaceId: ctx.workspace.id,
        externalId: input.externalId,
        error:
          err instanceof Error
            ? {
                name: err.name,
                message: err.message,
                stack: err.stack,
              }
            : String(err),
      });

      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message:
          "We encountered an issue while creating the identity. Our team has been notified. Please try again or contact support@unkey.com for assistance.",
      });
    }
  });

import { insertAuditLogs } from "@/lib/audit";
import { type Identity, db, schema } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { newId } from "@unkey/id";
import { z } from "zod";
import { workspaceProcedure } from "../../trpc";

export const createIdentityInputSchema = z.object({
  externalId: z
    .string()
    .min(1, "External ID is required")
    .max(255, "External ID is too long")
    .trim()
    .refine((id) => !/^\s+$/.test(id), "External ID cannot be only whitespace"),
  meta: z.record(z.unknown()).nullable(),
});

export const createIdentity = workspaceProcedure
  .input(createIdentityInputSchema)
  .mutation(async ({ input, ctx }) => {
    const identityId = newId("identity");

    try {
      // Check if identity with this externalId already exists
      const existingIdentity = await db.query.identities.findFirst({
        where: (table, { and, eq }) =>
          and(eq(table.workspaceId, ctx.workspace.id), eq(table.externalId, input.externalId)),
      });

      if (existingIdentity) {
        console.info({
          message: "Attempted to create duplicate identity",
          workspaceId: ctx.workspace.id,
          externalId: input.externalId,
          existingIdentityId: existingIdentity.id,
        });

        throw new TRPCError({
          code: "CONFLICT",
          message: "An identity with this external ID already exists in your workspace.",
        });
      }

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
        const payload: Identity = {
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
          "We encountered an issue while creating the identity. Our team has been notified. Please try again or contact support@unkey.dev for assistance.",
      });
    }
  });

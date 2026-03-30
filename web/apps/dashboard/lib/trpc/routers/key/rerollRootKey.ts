import { insertAuditLogs } from "@/lib/audit";
import { db, eq, schema } from "@/lib/db";
import { env } from "@/lib/env";
import { TRPCError } from "@trpc/server";
import { newId } from "@unkey/id";
import { newKey } from "@unkey/keys";
import { z } from "zod";
import { workspaceProcedure } from "../../trpc";

export const rerollRootKey = workspaceProcedure
  .input(
    z.object({
      keyId: z.string().min(1, "Key ID is required"),
      expiration: z
        .number()
        .int()
        .min(0, "Expiration must be non-negative")
        .max(4102444800000, "Expiration is too far in the future"),
    }),
  )
  .mutation(async ({ input, ctx }) => {
    // Root keys live in the UNKEY_WORKSPACE_ID but have forWorkspaceId pointing to user's workspace
    const existingKey = await db.query.keys
      .findFirst({
        where: (table, { and, eq, isNull }) =>
          and(
            eq(table.id, input.keyId),
            eq(table.forWorkspaceId, ctx.workspace.id),
            isNull(table.deletedAtM),
          ),
        with: {
          permissions: {
            columns: {
              permissionId: true,
            },
            with: {
              permission: {
                columns: {
                  id: true,
                  name: true,
                  slug: true,
                },
              },
            },
          },
        },
      })
      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "We were unable to find the root key. Please try again or contact support@unkey.com.",
        });
      });

    if (!existingKey) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "The specified root key was not found in your workspace.",
      });
    }

    // Root keys use the Unkey API's keyAuthId
    const unkeyApi = await db.query.apis
      .findFirst({
        where: (table, { and, eq }) =>
          and(
            eq(table.workspaceId, env().UNKEY_WORKSPACE_ID),
            eq(schema.apis.id, env().UNKEY_API_ID),
          ),
      })
      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "We were unable to find the Unkey API configuration. Please try again or contact support@unkey.com.",
        });
      });

    if (!unkeyApi) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: `API ${env().UNKEY_API_ID} was not found`,
      });
    }

    const keyAuthId = unkeyApi.keyAuthId;
    if (!keyAuthId) {
      throw new TRPCError({
        code: "PRECONDITION_FAILED",
        message: `API ${env().UNKEY_API_ID} is not setup to handle keys`,
      });
    }

    try {
      return await db.transaction(async (tx) => {
        const newKeyId = newId("key");
        const { key, hash, start } = await newKey({
          prefix: "unkey",
          byteLength: 16,
        });

        const now = Date.now();

        // Create the new root key
        await tx.insert(schema.keys).values({
          id: newKeyId,
          keyAuthId,
          name: existingKey.name,
          hash,
          start,
          ownerId: ctx.user.id,
          workspaceId: env().UNKEY_WORKSPACE_ID,
          forWorkspaceId: ctx.workspace.id,
          expires: null,
          createdAtM: now,
          updatedAtM: null,
          remaining: null,
          refillAmount: null,
          refillDay: null,
          lastRefillAt: null,
          enabled: true,
          deletedAtM: null,
        });

        // Copy permission assignments
        if (existingKey.permissions.length > 0) {
          await tx.insert(schema.keysPermissions).values(
            existingKey.permissions.map((p) => ({
              keyId: newKeyId,
              permissionId: p.permissionId,
              workspaceId: env().UNKEY_WORKSPACE_ID,
            })),
          );
        }

        // Set expiration on the old key
        const expiration =
          input.expiration === 0 ? new Date() : new Date(Date.now() + input.expiration);

        await tx
          .update(schema.keys)
          .set({
            expires: expiration,
            updatedAtM: now,
          })
          .where(eq(schema.keys.id, input.keyId));

        // Audit log
        await insertAuditLogs(tx, {
          workspaceId: ctx.workspace.id,
          actor: { type: "user", id: ctx.user.id },
          event: "key.reroll",
          description: `Rerolled root key ${input.keyId} to ${newKeyId}`,
          resources: [
            {
              type: "key",
              id: newKeyId,
              name: existingKey.name ?? undefined,
            },
            {
              type: "key",
              id: input.keyId,
              name: existingKey.name ?? undefined,
            },
          ],
          context: {
            location: ctx.audit.location,
            userAgent: ctx.audit.userAgent,
          },
        });

        return { keyId: newKeyId, key };
      });
    } catch (err) {
      if (err instanceof TRPCError) {
        throw err;
      }
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message:
          "We were unable to reroll the root key. Please try again or contact support@unkey.com",
      });
    }
  });

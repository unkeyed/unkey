import { insertAuditLogs } from "@/lib/audit";
import { db, eq, schema } from "@/lib/db";
import { env } from "@/lib/env";
import { Vault } from "@/lib/vault";
import { TRPCError } from "@trpc/server";
import { newId } from "@unkey/id";
import { newKey } from "@unkey/keys";
import { z } from "zod";
import { workspaceProcedure } from "../../trpc";

const vault = new Vault({
  baseUrl: env().VAULT_URL,
  token: env().VAULT_TOKEN,
});

export const rerollKey = workspaceProcedure
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
    const existingKey = await db.query.keys
      .findFirst({
        where: (table, { and, eq, isNull }) =>
          and(
            eq(table.id, input.keyId),
            eq(table.workspaceId, ctx.workspace.id),
            isNull(table.deletedAtM),
          ),
        with: {
          keyAuth: {
            with: {
              api: true,
            },
          },
          roles: {
            columns: {
              roleId: true,
            },
          },
          permissions: {
            columns: {
              permissionId: true,
            },
          },
          encrypted: true,
          ratelimits: true,
        },
      })
      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message: "We were unable to find the key. Please try again or contact support@unkey.com.",
        });
      });

    if (!existingKey) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "The specified key was not found in your workspace.",
      });
    }

    if (!existingKey.keyAuth) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "The key's API configuration could not be found.",
      });
    }

    // Extract prefix from existing key start
    let prefix = "";
    const split = existingKey.start.split("_");
    if (split.length > 1) {
      prefix = split[0];
    }

    // Fallback to API default prefix
    if (!prefix && existingKey.keyAuth.defaultPrefix) {
      prefix = existingKey.keyAuth.defaultPrefix;
    }

    const byteLength = existingKey.keyAuth.defaultBytes ?? 16;

    try {
      return await db.transaction(async (tx) => {
        const newKeyId = newId("key");
        const { key, hash, start } = await newKey({
          prefix: prefix || undefined,
          byteLength,
        });

        const now = Date.now();

        // Create the new key with all config from the original
        await tx.insert(schema.keys).values({
          id: newKeyId,
          keyAuthId: existingKey.keyAuthId,
          name: existingKey.name,
          hash,
          start,
          identityId: existingKey.identityId,
          ownerId: existingKey.ownerId,
          meta: existingKey.meta,
          workspaceId: ctx.workspace.id,
          forWorkspaceId: existingKey.forWorkspaceId,
          expires: existingKey.expires,
          createdAtM: now,
          updatedAtM: null,
          remaining: existingKey.remaining,
          refillDay: existingKey.refillDay,
          refillAmount: existingKey.refillAmount,
          lastRefillAt: existingKey.lastRefillAt,
          enabled: existingKey.enabled,
          environment: existingKey.environment,
        });

        // Copy encrypted key if the original was recoverable
        if (existingKey.encrypted) {
          const { encrypted, keyId: encryptionKeyId } = await vault.encrypt({
            keyring: ctx.workspace.id,
            data: key,
          });

          await tx.insert(schema.encryptedKeys).values({
            encrypted,
            encryptionKeyId,
            keyId: newKeyId,
            workspaceId: ctx.workspace.id,
            createdAt: now,
            updatedAt: null,
          });
        }

        // Copy ratelimits
        if (existingKey.ratelimits.length > 0) {
          await tx.insert(schema.ratelimits).values(
            existingKey.ratelimits.map((rl) => ({
              id: newId("ratelimit"),
              keyId: newKeyId,
              duration: rl.duration,
              limit: rl.limit,
              name: rl.name,
              workspaceId: ctx.workspace.id,
              createdAt: now,
              updatedAt: null,
              autoApply: rl.autoApply,
            })),
          );
        }

        // Copy role assignments
        if (existingKey.roles.length > 0) {
          await tx.insert(schema.keysRoles).values(
            existingKey.roles.map((r) => ({
              keyId: newKeyId,
              roleId: r.roleId,
              workspaceId: ctx.workspace.id,
              createdAtM: now,
            })),
          );
        }

        // Copy permission assignments
        if (existingKey.permissions.length > 0) {
          await tx.insert(schema.keysPermissions).values(
            existingKey.permissions.map((p) => ({
              keyId: newKeyId,
              permissionId: p.permissionId,
              workspaceId: ctx.workspace.id,
              createdAt: now,
              updatedAt: null,
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
          description: `Rerolled key ${input.keyId} to ${newKeyId}`,
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
        message: "We were unable to reroll the key. Please try again or contact support@unkey.com.",
      });
    }
  });

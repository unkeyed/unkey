import { type UnkeyAuditLog, insertAuditLogs } from "@/lib/audit";
import { and, db, eq, isNull, schema } from "@/lib/db";
import { env } from "@/lib/env";
import { Vault } from "@/lib/vault";
import { TRPCError } from "@trpc/server";
import { newId } from "@unkey/id";
import { newKey } from "@unkey/keys";
import { z } from "zod";
import { ratelimit, withRatelimit, workspaceProcedure } from "../../trpc";
import { roundUpToNextMinute } from "./reroll/round-up-to-next-minute";

const vault = new Vault({
  baseUrl: env().VAULT_URL,
  token: env().VAULT_TOKEN,
});

const rerollInputSchema = z.object({
  keyId: z.string().min(3).max(255),
  expiration: z.number().int().min(0).max(4_102_444_800_000),
});

export const rerollKey = workspaceProcedure
  .use(withRatelimit(ratelimit.create))
  .input(rerollInputSchema)
  .mutation(async ({ input, ctx }) => {
    return rerollKeyCore({
      keyId: input.keyId,
      expiration: input.expiration,
      scopedWorkspaceId: ctx.workspace.id,
      ctx,
    });
  });

export const rerollRootKey = workspaceProcedure
  .use(withRatelimit(ratelimit.create))
  .input(rerollInputSchema)
  .mutation(async ({ input, ctx }) => {
    return rerollKeyCore({
      keyId: input.keyId,
      expiration: input.expiration,
      scopedWorkspaceId: env().UNKEY_WORKSPACE_ID,
      forWorkspaceId: ctx.workspace.id,
      ctx,
    });
  });

type RerollKeyContext = {
  workspace: { id: string };
  user: { id: string };
  audit: {
    location: string;
    userAgent?: string;
  };
};

type RerollKeyArgs = {
  keyId: string;
  expiration: number;
  scopedWorkspaceId: string;
  forWorkspaceId?: string;
  ctx: RerollKeyContext;
};

async function rerollKeyCore({
  keyId,
  expiration,
  scopedWorkspaceId,
  forWorkspaceId,
  ctx,
}: RerollKeyArgs) {
  const source = await db.query.keys
    .findFirst({
      where: (table, { and, eq, isNull }) => {
        const clauses = [
          eq(table.workspaceId, scopedWorkspaceId),
          eq(table.id, keyId),
          isNull(table.deletedAtM),
        ];
        if (forWorkspaceId) {
          clauses.push(eq(table.forWorkspaceId, forWorkspaceId));
        }
        return and(...clauses);
      },
      with: {
        keyAuth: {
          with: {
            api: true,
          },
        },
        encrypted: true,
        ratelimits: true,
        roles: true,
        permissions: true,
      },
    })
    .catch((_err) => {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "We were unable to rotate this key. Please try again or contact support@unkey.com",
      });
    });

  if (!source) {
    throw new TRPCError({
      code: "NOT_FOUND",
      message: "The specified key was not found.",
    });
  }

  if (source.encrypted && !source.keyAuth.storeEncryptedKeys) {
    throw new TRPCError({
      code: "PRECONDITION_FAILED",
      message: "This API does not support key encryption.",
    });
  }

  const prefixFromStart = source.start.includes("_") ? source.start.split("_")[0] : "";
  const prefix = prefixFromStart || source.keyAuth.defaultPrefix || "";
  const byteLength = source.keyAuth.defaultBytes ?? 16;

  const { key: plaintext, hash, start } = await newKey({ prefix, byteLength });
  const newKeyId = newId("key");
  const now = Date.now();
  const oldKeyExpiresAt =
    expiration === 0 ? new Date(now) : roundUpToNextMinute(new Date(now + expiration));
  const shouldEncrypt = Boolean(source.encrypted);

  try {
    await db.transaction(async (tx) => {
      await tx.insert(schema.keys).values({
        id: newKeyId,
        keyAuthId: source.keyAuthId,
        hash,
        start,
        workspaceId: source.workspaceId,
        forWorkspaceId: source.forWorkspaceId,
        name: source.name,
        ownerId: source.ownerId,
        identityId: source.identityId,
        meta: source.meta,
        expires: null,
        createdAtM: now,
        refillDay: source.refillDay,
        refillAmount: source.refillAmount,
        lastRefillAt: source.refillAmount != null ? new Date(now) : null,
        enabled: source.enabled,
        remaining: source.remaining,
        environment: source.environment,
      });

      if (shouldEncrypt) {
        const { encrypted, keyId: encryptionKeyId } = await vault.encrypt({
          keyring: source.workspaceId,
          data: plaintext,
        });
        await tx.insert(schema.encryptedKeys).values({
          workspaceId: source.workspaceId,
          keyId: newKeyId,
          encrypted,
          encryptionKeyId,
          createdAt: now,
        });
      }

      const keyScopedRatelimits = source.ratelimits.filter((rl) => rl.keyId && !rl.identityId);
      if (keyScopedRatelimits.length > 0) {
        await tx.insert(schema.ratelimits).values(
          keyScopedRatelimits.map((rl) => ({
            id: newId("ratelimit"),
            workspaceId: source.workspaceId,
            keyId: newKeyId,
            name: rl.name,
            limit: rl.limit,
            duration: rl.duration,
            autoApply: rl.autoApply,
            createdAt: now,
          })),
        );
      }

      if (source.roles.length > 0) {
        await tx.insert(schema.keysRoles).values(
          source.roles.map((role) => ({
            keyId: newKeyId,
            roleId: role.roleId,
            workspaceId: source.workspaceId,
            createdAtM: now,
          })),
        );
      }

      if (source.permissions.length > 0) {
        await tx.insert(schema.keysPermissions).values(
          source.permissions.map((permission) => ({
            keyId: newKeyId,
            permissionId: permission.permissionId,
            workspaceId: source.workspaceId,
            createdAtM: now,
          })),
        );
      }

      await tx
        .update(schema.keys)
        .set({ expires: oldKeyExpiresAt })
        .where(and(eq(schema.keys.id, source.id), isNull(schema.keys.deletedAtM)));

      const resources: UnkeyAuditLog["resources"] = [
        { type: "key", id: newKeyId, name: source.name ?? undefined },
        { type: "key", id: source.id, name: source.name ?? undefined },
      ];
      if (source.keyAuth.api) {
        resources.push({
          type: "api",
          id: source.keyAuth.api.id,
          name: source.keyAuth.api.name,
        });
      }

      await insertAuditLogs(tx, {
        workspaceId: ctx.workspace.id,
        actor: { type: "user", id: ctx.user.id },
        event: "key.reroll",
        description: `Rerolled key (${source.id}) to (${newKeyId})`,
        resources,
        context: {
          location: ctx.audit.location,
          userAgent: ctx.audit.userAgent,
        },
      });
    });
  } catch (err) {
    if (err instanceof TRPCError) {
      throw err;
    }
    throw new TRPCError({
      code: "INTERNAL_SERVER_ERROR",
      message: "We were unable to rotate this key. Please try again or contact support@unkey.com",
    });
  }

  return {
    keyId: newKeyId,
    key: plaintext,
    name: source.name ?? undefined,
  };
}

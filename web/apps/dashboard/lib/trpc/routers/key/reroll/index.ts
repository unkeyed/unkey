import {
  GRACE_PERIOD_VALUES_MS,
  type GracePeriodMs,
} from "@/components/api-keys-table/components/actions/components/rotate-key/rotate-key.constants";
import { type UnkeyAuditLog, insertAuditLogs } from "@/lib/audit";
import { and, db, eq, isNull, schema } from "@/lib/db";
import { env } from "@/lib/env";
import { Vault } from "@/lib/vault";
import { TRPCError } from "@trpc/server";
import { newId } from "@unkey/id";
import { newKey } from "@unkey/keys";
import { z } from "zod";
import { ratelimit, withRatelimit, workspaceProcedure } from "../../../trpc";
import { capGracePeriodAtSourceExpiry } from "./cap-grace-period-at-source-expiry";
import { roundUpToNextMinute } from "./round-up-to-next-minute";

const vault = new Vault({
  baseUrl: env().VAULT_URL,
  token: env().VAULT_TOKEN,
});

const allowedExpirations = new Set<number>(GRACE_PERIOD_VALUES_MS);

const rerollInputSchema = z.object({
  keyId: z.string().min(3).max(255),
  expiration: z
    .number()
    .int()
    .refine((v): v is GracePeriodMs => allowedExpirations.has(v), {
      error: "expiration must be one of the supported grace periods",
    }),
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
    .catch((err) => {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "We were unable to rotate this key. Please try again or contact support@unkey.com",
        cause: err,
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

  const now = Date.now();

  // Refuse to rotate an already-expired key. The new key inherits the
  // source's expiry, so rotating an expired key would mint a replacement
  // that is born already expired — almost certainly not what the caller
  // intended. This outer check is a fast-fail; the in-transaction re-read
  // below is authoritative against concurrent expiry updates.
  if (source.expires && source.expires.getTime() <= now) {
    throw new TRPCError({
      code: "PRECONDITION_FAILED",
      message: "This key has already expired and cannot be rotated.",
    });
  }

  // Preserve the source key's prefix exactly. Falling back to the current
  // keyAuth default would silently add a prefix when the default has been
  // changed after the key was created. Prefixes may contain underscores
  // (e.g. "pk_test"), so we split on the *last* underscore — the base58
  // alphabet has no `_`, so every underscore in `start` came from a prefix
  // separator.
  const lastUnderscore = source.start.lastIndexOf("_");
  const prefix = lastUnderscore === -1 ? "" : source.start.slice(0, lastUnderscore);

  // Byte length falls back to the workspace's current default because the
  // original per-key length was never persisted (no column on `keys`). If
  // an admin changed `keyAuth.defaultBytes` after the source key was
  // created, the rotated key will use the new length. Preserving the
  // exact original length would require a schema migration to record it
  // at creation time.
  const byteLength = source.keyAuth.defaultBytes ?? 16;

  const { key: plaintext, hash, start } = await newKey({ prefix, byteLength });
  const newKeyId = newId("key");
  const gracePeriodEnd =
    expiration === 0 ? new Date(now) : roundUpToNextMinute(new Date(now + expiration));
  const shouldEncrypt = Boolean(source.encrypted);

  try {
    // Encrypt outside the transaction. vault.encrypt is a network RPC; doing
    // it inside the transaction would hold row locks for its duration. The
    // call is stateless — if the transaction later fails, the ciphertext is
    // simply discarded, nothing to clean up vault-side.
    const encryptedRecord = shouldEncrypt
      ? await vault.encrypt({ keyring: source.workspaceId, data: plaintext })
      : undefined;

    await db.transaction(async (tx) => {
      // Re-check the source's state inside the transaction. The outer read
      // was a snapshot; without this a concurrent delete or expiration-update
      // that lands before the transaction starts could let us revive a
      // soft-deleted key or resurrect one that just expired. Uses a fresh
      // timestamp — the outer `now` was captured before the transaction,
      // so a key whose expiry falls in that gap wouldn't be caught.
      const nowInTx = Date.now();
      const current = await tx.query.keys.findFirst({
        where: (table, { and, eq }) =>
          and(eq(table.workspaceId, source.workspaceId), eq(table.id, source.id)),
        columns: { expires: true, deletedAtM: true },
      });
      if (!current || current.deletedAtM !== null) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "The specified key was not found.",
        });
      }
      if (current.expires && current.expires.getTime() <= nowInTx) {
        throw new TRPCError({
          code: "PRECONDITION_FAILED",
          message: "This key has already expired and cannot be rotated.",
        });
      }

      // Re-check the keyAuth's `storeEncryptedKeys` flag inside the tx.
      // The outer guard used the joined snapshot from before the tx; if
      // an admin flipped encryption off in the gap, committing a new
      // encryptedKeys row here would bypass that policy. The source key
      // itself is already encrypted — rotation has no safe fallback, so
      // refuse and let the caller resolve the policy change.
      if (source.encrypted) {
        const currentKeyAuth = await tx.query.keyAuth.findFirst({
          where: (table, { eq }) => eq(table.id, source.keyAuthId),
          columns: { storeEncryptedKeys: true },
        });
        if (!currentKeyAuth || !currentKeyAuth.storeEncryptedKeys) {
          throw new TRPCError({
            code: "PRECONDITION_FAILED",
            message: "This API does not support key encryption.",
          });
        }
      }

      // The new key inherits the source's expiry so rotation preserves the
      // original lifetime constraint instead of silently producing a
      // permanent key. The old key's grace period is capped against that
      // same expiry. Both the cap and the new-key insert use
      // `current.expires` (re-read inside this tx) rather than the outer
      // `source.expires` snapshot, so any concurrent expiry update that
      // landed before this transaction is respected.
      const oldKeyExpiresAt = capGracePeriodAtSourceExpiry(current.expires, gracePeriodEnd);

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
        expires: current.expires,
        createdAtM: now,
        refillDay: source.refillDay,
        refillAmount: source.refillAmount,
        // Inherit the source's last-refill timestamp rather than stamping
        // `now`. The new key carries over `source.remaining`, so those
        // credits were genuinely last topped up at `source.lastRefillAt`;
        // resetting the timestamp would misreport the balance's age.
        lastRefillAt: source.refillAmount != null ? source.lastRefillAt : null,
        enabled: source.enabled,
        remaining: source.remaining,
        environment: source.environment,
      });

      if (encryptedRecord) {
        await tx.insert(schema.encryptedKeys).values({
          workspaceId: source.workspaceId,
          keyId: newKeyId,
          encrypted: encryptedRecord.encrypted,
          encryptionKeyId: encryptedRecord.keyId,
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

      // Guard against a concurrent soft-delete landing between the in-tx
      // re-check and this update. The WHERE clause includes `deletedAtM IS
      // NULL`, so if the source row was deleted in the gap the update
      // matches zero rows and the old key's expiry never gets capped.
      // Abort the transaction in that case so we don't commit a new key
      // tied to a now-deleted source and an audit entry that claims an
      // expiry we didn't actually set.
      const updateResult = await tx
        .update(schema.keys)
        .set({ expires: oldKeyExpiresAt })
        .where(
          and(
            eq(schema.keys.id, source.id),
            eq(schema.keys.workspaceId, source.workspaceId),
            isNull(schema.keys.deletedAtM),
          ),
        );
      if (updateResult[0].affectedRows === 0) {
        throw new TRPCError({
          code: "PRECONDITION_FAILED",
          message: "This key was deleted while rotating. The rotation has been cancelled.",
        });
      }

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
        description: `Rerolled key (${source.id}) to (${newKeyId}); old key expires at ${oldKeyExpiresAt.toISOString()}`,
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
      cause: err,
    });
  }

  return {
    keyId: newKeyId,
    key: plaintext,
    name: source.name ?? undefined,
  };
}

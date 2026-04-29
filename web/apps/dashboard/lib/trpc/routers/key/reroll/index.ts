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

// Rotates a regular API key owned by the caller's workspace. The new key
// inherits the source's metadata and authorization scope; the old key's
// expiry is shortened to `now + expiration` (capped at the source's own
// expiry so rotation never extends a key's lifetime).
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

// Rotates a root key. Root keys live in the Unkey-owned workspace
// (`UNKEY_WORKSPACE_ID`) but are bound to a customer workspace via
// `forWorkspaceId`. The lookup is scoped by both: the row must belong to
// the Unkey workspace AND `forWorkspaceId` must match the caller's
// workspace, so a tenant can only rotate their own root keys.
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
  const newKeyId = newId("key");
  const now = Date.now();
  const gracePeriodEnd = new Date(now + expiration);

  try {
    return await db.transaction(async (tx) => {
      // Lock the source row before reading anything we copy onto the new
      // key. drizzle 0.45.1's relational `findFirst` doesn't expose
      // `.for("update")`, so we lock with a core query carrying the full
      // filter set, then fetch relations by id. Applying every filter
      // here means the relational read can't disagree with the lock —
      // any row we lock is one we're committed to using.
      const lockFilters = [
        eq(schema.keys.workspaceId, scopedWorkspaceId),
        eq(schema.keys.id, keyId),
        isNull(schema.keys.deletedAtM),
      ];
      if (forWorkspaceId) {
        lockFilters.push(eq(schema.keys.forWorkspaceId, forWorkspaceId));
      }
      const [locked] = await tx
        .select({ id: schema.keys.id })
        .from(schema.keys)
        .where(and(...lockFilters))
        .for("update");
      if (!locked) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "The specified key was not found.",
        });
      }

      const source = await tx.query.keys.findFirst({
        where: (table, { eq }) => eq(table.id, keyId),
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
      });

      if (!source) {
        // Unreachable: the FOR UPDATE above guarantees the row exists.
        // Present only to narrow the type for the rest of the tx.
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "We were unable to rotate this key. Please try again or contact support@unkey.com",
        });
      }

      const nowInTx = Date.now();
      if (source.expires && source.expires.getTime() <= nowInTx) {
        throw new TRPCError({
          code: "PRECONDITION_FAILED",
          message: "This key has already expired and cannot be rotated.",
        });
      }

      // The source key is encrypted, so the rotated key must be too —
      // there's no safe fallback. If `storeEncryptedKeys` was flipped
      // off after the source was created, refuse rather than commit a
      // new encryptedKeys row that bypasses the current policy.
      if (source.encrypted && !source.keyAuth.storeEncryptedKeys) {
        throw new TRPCError({
          code: "PRECONDITION_FAILED",
          message: "This API does not support key encryption.",
        });
      }

      // Preserve the source key's prefix exactly. Falling back to the
      // current keyAuth default would silently add a prefix when the
      // default has been changed after the key was created. Prefixes may
      // contain underscores (e.g. "pk_test"), so we split on the *last*
      // underscore — the base58 alphabet has no `_`, so every underscore
      // in `start` came from a prefix separator.
      const lastUnderscore = source.start.lastIndexOf("_");
      const prefix = lastUnderscore === -1 ? "" : source.start.slice(0, lastUnderscore);

      // Byte length falls back to the workspace's current default because
      // the original per-key length was never persisted (no column on
      // `keys`). If an admin changed `keyAuth.defaultBytes` after the
      // source key was created, the rotated key will use the new length.
      // Preserving the exact original length would require a schema
      // migration to record it at creation time.
      const byteLength = source.keyAuth.defaultBytes ?? 16;

      const { key: plaintext, hash, start } = await newKey({ prefix, byteLength });

      // Encrypt inside the tx so the decision uses the locked source
      // state. The lock is held for the duration of this RPC, but key
      // rotation is rare enough that this is acceptable; the alternative
      // (speculative pre-tx encrypt) reintroduces a TOCTOU on the
      // `encrypted` flag and the keyring.
      const encryptedRecord = source.encrypted
        ? await vault.encrypt({ keyring: source.workspaceId, data: plaintext })
        : undefined;

      // The new key inherits the source's expiry so rotation preserves
      // the original lifetime constraint instead of silently producing a
      // permanent key. The old key's grace period is capped against that
      // same expiry.
      const oldKeyExpiresAt = capGracePeriodAtSourceExpiry(source.expires, gracePeriodEnd);

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
        expires: source.expires,
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

      // Skip the UPDATE when the cap is a no-op (gracePeriodEnd >= source's
      // existing expiry, so capGracePeriodAtSourceExpiry returns the source's
      // current value unchanged). mysql2 reports affectedRows as rows
      // *changed*, not matched, so a no-op write would falsely look like a
      // missing row. The FOR UPDATE lock above keeps the source row stable
      // for the rest of this tx, so we don't need the WHERE-with-deletedAtM
      // pattern here for soft-delete detection.
      if (source.expires?.getTime() !== oldKeyExpiresAt.getTime()) {
        await tx
          .update(schema.keys)
          .set({ expires: oldKeyExpiresAt })
          .where(
            and(eq(schema.keys.id, source.id), eq(schema.keys.workspaceId, source.workspaceId)),
          );
      }

      const resources: UnkeyAuditLog["resources"] = [
        { type: "key", id: newKeyId, name: source.name ?? undefined },
        {
          type: "key",
          id: source.id,
          name: source.name ?? undefined,
          meta: { expiresAt: oldKeyExpiresAt.getTime() },
        },
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

      return {
        keyId: newKeyId,
        key: plaintext,
        name: source.name ?? undefined,
      };
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
}

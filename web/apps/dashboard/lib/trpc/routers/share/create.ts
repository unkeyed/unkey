import { randomBytes } from "node:crypto";
import { VaultService } from "@/gen/proto/vault/v1/service_pb";
import { insertAuditLogs } from "@/lib/audit";
import { db, lt, schema } from "@/lib/db";
import { createVaultClient } from "@/lib/vault-client";
import { z } from "zod";
import { withCreateRateLimit, workspaceProcedure } from "../../trpc";

const vault = createVaultClient(VaultService);

// Share links always live for 72 hours, enforced server-side.
const TTL_MS = 72 * 60 * 60 * 1000;

// Keeps the vault ciphertext within the `encrypted` varchar(1024) column. Keys
// are far shorter; this just rejects oversized input cleanly.
const MAX_SECRET_LENGTH = 512;

// Vault-encrypts the secret (keyring = workspace id, like `encrypted_keys`) and
// stores it for a one-time share link.
export const createSharedSecret = workspaceProcedure
  .use(withCreateRateLimit())
  .input(z.object({ secret: z.string().min(1).max(MAX_SECRET_LENGTH) }))
  .mutation(async ({ ctx, input }) => {
    const { encrypted, keyId } = await vault.encrypt({
      keyring: ctx.workspace.id,
      data: input.secret,
    });

    const now = Date.now();

    // Lazy cleanup: MySQL has no native row TTL, so sweep expired rows on write.
    // The read path enforces expiry independently, so this is only housekeeping.
    await db.delete(schema.sharedSecrets).where(lt(schema.sharedSecrets.expiresAt, now));

    // The id is a bearer credential in the URL, so use raw randomness with no
    // prefix that would advertise what it points at.
    const id = randomBytes(16).toString("base64url");

    // Insert and audit atomically so a failed audit can't leave an unrecorded row.
    await db.transaction(async (tx) => {
      await tx.insert(schema.sharedSecrets).values({
        id,
        workspaceId: ctx.workspace.id,
        expiresAt: now + TTL_MS,
        encrypted,
        encryptionKeyId: keyId,
      });

      await insertAuditLogs(tx, {
        workspaceId: ctx.workspace.id,
        actor: { type: "user", id: ctx.user.id },
        event: "secret.create",
        description: `Created a one-time share link ${id}`,
        resources: [{ type: "secret", id }],
        context: { location: ctx.audit.location, userAgent: ctx.audit.userAgent },
      });
    });

    return { id };
  });

import { VaultService } from "@/gen/proto/vault/v1/service_pb";
import { insertAuditLogs } from "@/lib/audit";
import { db, eq, schema } from "@/lib/db";
import { createVaultClient } from "@/lib/vault-client";
import { z } from "zod";
import { publicProcedure } from "../../trpc";

const vault = createVaultClient(VaultService);

// Single result shape for every "can't reveal" case (missing / expired / already
// used). Distinguishing them would leak whether an id ever existed.
type RevealResult = { ok: true; secret: string } | { ok: false };

// Public: the share id is the bearer credential. The whole read -> decrypt ->
// delete -> audit sequence runs in one transaction. `SELECT ... FOR UPDATE` locks
// the row so concurrent reveals of the same id serialize: the loser blocks until
// we commit, then reads nothing. Decrypt happens before the delete, so a vault
// outage rolls the transaction back and leaves the row retryable.
export const revealSharedSecret = publicProcedure
  .input(z.object({ id: z.string().min(1).max(256) }))
  .mutation(async ({ ctx, input }): Promise<RevealResult> => {
    return db.transaction(async (tx): Promise<RevealResult> => {
      const [row] = await tx
        .select()
        .from(schema.sharedSecrets)
        .where(eq(schema.sharedSecrets.id, input.id))
        .for("update");

      if (!row || row.expiresAt <= Date.now()) {
        return { ok: false };
      }

      const { plaintext } = await vault.decrypt({
        keyring: row.workspaceId,
        encrypted: row.encrypted,
      });

      await tx.delete(schema.sharedSecrets).where(eq(schema.sharedSecrets.id, input.id));

      await insertAuditLogs(tx, {
        workspaceId: row.workspaceId,
        actor: { type: "system", id: "secret-share" },
        event: "secret.decrypt",
        description: "One-time share link was revealed",
        resources: [],
        context: { location: ctx.audit.location, userAgent: ctx.audit.userAgent },
      });

      return { ok: true, secret: plaintext };
    });
  });

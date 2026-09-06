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

// Decrypt speculatively without holding a database connection. Only the caller
// that locks and consumes the same, still-valid row may return the plaintext.
export const revealSharedSecret = publicProcedure
  .input(z.object({ id: z.string().min(1).max(256) }))
  .mutation(async ({ ctx, input }): Promise<RevealResult> => {
    const [candidate] = await db
      .select({
        workspaceId: schema.sharedSecrets.workspaceId,
        expiresAt: schema.sharedSecrets.expiresAt,
        encrypted: schema.sharedSecrets.encrypted,
      })
      .from(schema.sharedSecrets)
      .where(eq(schema.sharedSecrets.id, input.id));
    if (!candidate || candidate.expiresAt <= Date.now()) {
      return { ok: false };
    }
    const { plaintext } = await vault.decrypt({
      keyring: candidate.workspaceId,
      encrypted: candidate.encrypted,
    });

    return db.transaction(async (tx): Promise<RevealResult> => {
      const [row] = await tx
        .select({
          workspaceId: schema.sharedSecrets.workspaceId,
          expiresAt: schema.sharedSecrets.expiresAt,
          encrypted: schema.sharedSecrets.encrypted,
        })
        .from(schema.sharedSecrets)
        .where(eq(schema.sharedSecrets.id, input.id))
        .for("update");

      if (
        !row ||
        row.expiresAt <= Date.now() ||
        row.workspaceId !== candidate.workspaceId ||
        row.encrypted !== candidate.encrypted ||
        row.expiresAt !== candidate.expiresAt
      ) {
        return { ok: false };
      }

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

import { and, db, eq, isNull, schema } from "@/lib/db";
import { env } from "@/lib/env";
import { Vault } from "@/lib/vault";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { workspaceProcedure } from "../../../trpc";
import { drainConfigSchema, environmentSchema, filtersSchema, sourceSchema } from "./schemas";

const vault = new Vault({
  baseUrl: env().VAULT_URL,
  token: env().VAULT_TOKEN,
});

// Update accepts the same shape as create but with an id. Credential
// rotation is opt-in: caller passes the field only when rotating, so a
// no-op edit (e.g. renaming the drain) does not invalidate the cached
// plaintext in the coordinator.
const inputSchema = z
  .object({
    id: z.string(),
    name: z.string().trim().min(1).max(256),
    sources: z.array(sourceSchema).min(1),
    environments: z.array(environmentSchema).min(1),
    apps: z.array(z.string()),
    filters: filtersSchema,
    credential: z.string().trim().min(1).optional(),
  })
  .and(drainConfigSchema);

export const updateLogDrain = workspaceProcedure
  .input(inputSchema)
  .mutation(async ({ ctx, input }) => {
    const now = Date.now();

    try {
      // Confirm ownership before touching the row. Selecting the existing
      // provider lets us reject a provider switch — that is a credential
      // and config-shape rotation that should be a delete + recreate, not
      // an in-place update.
      const existing = await db.query.logDrains.findFirst({
        where: and(
          eq(schema.logDrains.id, input.id),
          eq(schema.logDrains.workspaceId, ctx.workspace.id as string),
          isNull(schema.logDrains.deletedAt),
        ),
        columns: { id: true, provider: true },
      });
      if (!existing) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "Log drain not found",
        });
      }
      if (existing.provider !== input.provider) {
        throw new TRPCError({
          code: "BAD_REQUEST",
          message: "Cannot change provider on an existing drain. Delete and recreate.",
        });
      }

      await db.transaction(async (tx) => {
        await tx
          .update(schema.logDrains)
          .set({
            name: input.name,
            config: input.config,
            sources: input.sources,
            environments: input.environments,
            apps: input.apps,
            filters: input.filters,
            updatedAt: now,
          })
          .where(eq(schema.logDrains.id, input.id));

        if (input.credential) {
          const { encrypted, keyId } = await vault.encrypt({
            keyring: ctx.workspace.id as string,
            data: input.credential,
          });
          await tx
            .update(schema.logDrainCredentials)
            .set({
              source: "paste",
              encryptedCredentials: encrypted,
              encryptionKeyId: keyId,
              oauthGrantId: null,
              updatedAt: now,
            })
            .where(eq(schema.logDrainCredentials.drainId, input.id));
        }
      });
      return { id: input.id };
    } catch (error) {
      if (error instanceof TRPCError) {
        throw error;
      }
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to update log drain",
      });
    }
  });

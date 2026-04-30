import { db, schema } from "@/lib/db";
import { env } from "@/lib/env";
import { Vault } from "@/lib/vault";
import { TRPCError } from "@trpc/server";
import { newId } from "@unkey/id";
import { z } from "zod";
import { workspaceProcedure } from "../../../trpc";
import { drainConfigSchema, environmentSchema, filtersSchema, sourceSchema } from "./schemas";

const vault = new Vault({
  baseUrl: env().VAULT_URL,
  token: env().VAULT_TOKEN,
});

const inputSchema = z
  .object({
    name: z.string().trim().min(1).max(256),

    // Either project-scoped (UUID) or workspace-scoped (null = catch-all).
    projectId: z.string().nullable(),

    sources: z.array(sourceSchema).min(1).default(["runtime", "request"]),
    environments: z.array(environmentSchema).min(1).default(["production"]),
    apps: z.array(z.string()).default([]),
    filters: filtersSchema,

    // OAuth-source drains are deferred to the integrations stack — for now
    // every create on this router supplies a paste token. The credential
    // never round-trips back to the dashboard; the next response shows
    // only the masked form.
    credentialSource: z.literal("paste"),
    credential: z.string().trim().min(1),
  })
  // Discriminated union of provider + config sits at the top level so a
  // misshaped axiom payload fails validation before we touch Vault or DB.
  .and(drainConfigSchema);

export const createLogDrain = workspaceProcedure
  .input(inputSchema)
  .mutation(async ({ ctx, input }) => {
    const id = newId("logDrain");
    const now = Date.now();

    try {
      // Encrypt the pasted token via Vault keyed on the workspace, matching
      // the convention env-vars and ACME certificates use. Re-encrypts
      // workspace-wide when KMS material rotates.
      const { encrypted, keyId } = await vault.encrypt({
        keyring: ctx.workspace.id as string,
        data: input.credential,
      });

      // Atomic insert across log_drains + log_drain_credentials. State and
      // cursor rows are created lazily by the coordinator so we never
      // bootstrap them with stale defaults that the coordinator would
      // immediately overwrite.
      await db.transaction(async (tx) => {
        await tx.insert(schema.logDrains).values({
          id,
          workspaceId: ctx.workspace.id as string,
          projectId: input.projectId,
          name: input.name,
          provider: input.provider,
          config: input.config,
          sources: input.sources,
          environments: input.environments,
          apps: input.apps,
          filters: input.filters,
          deliveryMode: "batch",
          enabled: true,
          createdAt: now,
        });

        await tx.insert(schema.logDrainCredentials).values({
          drainId: id,
          source: "paste",
          encryptedCredentials: encrypted,
          encryptionKeyId: keyId,
          oauthGrantId: null,
          updatedAt: now,
        });
      });

      // Test push lands in a follow-up router file so create stays narrow.
      // The dashboard wizard calls testPush after create on success.
      return { id };
    } catch (error) {
      if (error instanceof TRPCError) {
        throw error;
      }
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to create log drain",
      });
    }
  });

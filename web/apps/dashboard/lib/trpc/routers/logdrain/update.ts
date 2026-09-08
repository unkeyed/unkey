import { VaultService } from "@/gen/proto/vault/v1/service_pb";
import { insertAuditLogs } from "@/lib/audit";
import { and, db, eq, schema } from "@/lib/db";
import { createVaultClient } from "@/lib/vault-client";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { workspaceProcedure } from "../../trpc";
import {
  type EncryptedHttpHeader,
  decodeLogdrainConfig,
  encodeLogdrainConfig,
  encryptHttpHeaders,
} from "./config";
import {
  type HttpHeaderUpdate,
  httpFormatSchema,
  httpHeaderUpdatesSchema,
  httpsUrl,
} from "./validation";

const vault = createVaultClient(VaultService);
const updateDestinationSchema = z.discriminatedUnion("kind", [
  z.object({
    kind: z.literal("http"),
    config: z
      .object({
        url: httpsUrl.optional(),
        format: httpFormatSchema.optional(),
        headers: httpHeaderUpdatesSchema.optional(),
      })
      .refine(
        (config) =>
          config.url !== undefined || config.format !== undefined || config.headers !== undefined,
        "At least one HTTP destination update is required",
      ),
  }),
  z.object({
    kind: z.literal("axiom"),
    config: z
      .object({
        dataset: z.string().min(1).optional(),
        token: z.string().min(1).optional(),
      })
      .refine(
        (config) => config.dataset !== undefined || config.token !== undefined,
        "At least one Axiom destination update is required",
      ),
  }),
]);

export const updateLogdrain = workspaceProcedure
  .input(
    z
      .object({
        id: z.string().min(1),
        name: z.string().trim().min(1).max(128).optional(),
        status: z.enum(["running", "paused_by_user"]).optional(),
        destination: updateDestinationSchema.optional(),
      })
      .refine(
        (input) =>
          input.name !== undefined || input.status !== undefined || input.destination !== undefined,
        "At least one update is required",
      ),
  )
  .mutation(async ({ ctx, input }) => {
    try {
      const destination = input.destination;
      let encryptedHeaders: EncryptedHttpHeader[] | undefined;
      let encryptedToken: string | undefined;
      switch (destination?.kind) {
        case "http":
          if (destination.config.headers !== undefined) {
            const plaintextHeaders: Record<string, string> = {};
            for (const header of destination.config.headers) {
              if (header.mode === "set") {
                plaintextHeaders[header.name] = header.value;
              }
            }
            encryptedHeaders = await encryptHttpHeaders(ctx.workspace.id, plaintextHeaders);
          }
          break;
        case "axiom":
          if (destination.config.token !== undefined) {
            encryptedToken = (
              await vault.encrypt({ keyring: ctx.workspace.id, data: destination.config.token })
            ).encrypted;
          }
          break;
        case undefined:
          break;
        default:
          throw new Error(`Unsupported log drain sink: ${destination satisfies never}`);
      }
      await db.transaction(async (tx) => {
        const [drain] = await tx
          .select({
            id: schema.logdrains.id,
            name: schema.logdrains.name,
            config: schema.logdrains.config,
            status: schema.logdrains.status,
          })
          .from(schema.logdrains)
          .where(
            and(
              eq(schema.logdrains.id, input.id),
              eq(schema.logdrains.workspaceId, ctx.workspace.id),
            ),
          )
          .for("update");
        if (!drain) {
          throw new TRPCError({ code: "NOT_FOUND", message: "Log drain not found" });
        }
        const existing = decodeLogdrainConfig(drain.config);
        let config = drain.config;
        switch (destination?.kind) {
          case "http":
            switch (existing.kind) {
              case "http": {
                let headers = existing.headers;
                if (destination.config.headers !== undefined) {
                  if (encryptedHeaders === undefined) {
                    throw new Error("Encrypted HTTP headers are missing");
                  }
                  headers = applyHttpHeaderUpdates({
                    existing: existing.headers,
                    updates: destination.config.headers,
                    encrypted: encryptedHeaders,
                  });
                }
                config = encodeLogdrainConfig({
                  kind: destination.kind,
                  url: destination.config.url ?? existing.url,
                  format: destination.config.format ?? existing.format,
                  headers,
                });
                break;
              }
              case "axiom":
                throw new TRPCError({
                  code: "BAD_REQUEST",
                  message: "Kind cannot be changed. Create a new log drain instead.",
                });
              default:
                throw new Error(`Unsupported log drain sink: ${existing satisfies never}`);
            }
            break;
          case "axiom":
            switch (existing.kind) {
              case "http":
                throw new TRPCError({
                  code: "BAD_REQUEST",
                  message: "Kind cannot be changed. Create a new log drain instead.",
                });
              case "axiom":
                config = encodeLogdrainConfig({
                  kind: destination.kind,
                  dataset: destination.config.dataset ?? existing.dataset,
                  encryptedToken: encryptedToken ?? existing.encryptedToken,
                });
                break;
              default:
                throw new Error(`Unsupported log drain sink: ${existing satisfies never}`);
            }
            break;
          case undefined:
            break;
          default:
            throw new Error(`Unsupported log drain sink: ${destination satisfies never}`);
        }

        const resetFailureState = input.status === "running" || destination !== undefined;
        const expireLease = input.status !== undefined || destination !== undefined;
        const status =
          input.status ??
          (destination !== undefined && drain.status === "paused_by_failure"
            ? "running"
            : undefined);
        await tx
          .update(schema.logdrains)
          .set({
            ...(input.name !== undefined ? { name: input.name } : {}),
            ...(status !== undefined ? { status } : {}),
            ...(destination ? { config } : {}),
            // Expire the current lease so in-flight state writes fail and a
            // worker must acquire a new fencing token.
            ...(expireLease ? { leaseExpiresAt: 0 } : {}),
            ...(resetFailureState
              ? {
                  consecutiveFailures: 0,
                  nextAttemptAt: 0,
                }
              : {}),
          })
          .where(
            and(
              eq(schema.logdrains.id, input.id),
              eq(schema.logdrains.workspaceId, ctx.workspace.id),
            ),
          );
        await insertAuditLogs(tx, {
          workspaceId: ctx.workspace.id,
          actor: { type: "user", id: ctx.user.id },
          event: "logdrain.update",
          description: `Updated log drain ${input.id}`,
          resources: [{ type: "logdrain", id: drain.id, name: input.name ?? drain.name }],
          context: { location: ctx.audit.location, userAgent: ctx.audit.userAgent },
        });
      });
      return { id: input.id };
    } catch (error) {
      if (error instanceof TRPCError) {
        throw error;
      }
      console.error("Failed to update log drain", error);
      throw new TRPCError({ code: "INTERNAL_SERVER_ERROR", message: "Failed to update log drain" });
    }
  });

export function applyHttpHeaderUpdates({
  existing,
  updates,
  encrypted,
}: {
  existing: EncryptedHttpHeader[];
  updates: HttpHeaderUpdate[];
  encrypted: EncryptedHttpHeader[];
}): EncryptedHttpHeader[] {
  const existingByName = new Map(existing.map((header) => [header.name.toLowerCase(), header]));
  const encryptedByName = new Map(encrypted.map((header) => [header.name.toLowerCase(), header]));

  return updates.map((update) => {
    const normalizedName = update.name.toLowerCase();
    switch (update.mode) {
      case "preserve": {
        const header = existingByName.get(normalizedName);
        if (!header) {
          throw new TRPCError({
            code: "BAD_REQUEST",
            message: `Cannot preserve unknown HTTP header ${update.name}`,
          });
        }
        return header;
      }
      case "set": {
        const header = encryptedByName.get(normalizedName);
        if (!header) {
          throw new Error(`Encrypted HTTP header ${update.name} is missing`);
        }
        return { ...header, name: existingByName.get(normalizedName)?.name ?? header.name };
      }
    }
  });
}

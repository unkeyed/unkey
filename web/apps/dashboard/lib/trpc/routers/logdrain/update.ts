import { VaultService } from "@/gen/proto/vault/v1/service_pb";
import { insertAuditLogs } from "@/lib/audit";
import { and, db, eq, schema } from "@/lib/db";
import { createVaultClient } from "@/lib/vault-client";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { workspaceProcedure } from "../../trpc";
import {
  type EncryptedHttpHeader,
  type LogdrainConfig,
  decodeLogdrainConfig,
  encodeLogdrainConfig,
  encryptHttpHeaders,
} from "./config";
import {
  type HttpHeaderUpdate,
  eventTypesSchema,
  httpFormatSchema,
  httpHeaderUpdatesSchema,
  httpsUrl,
  identifiersSchema,
  keySpaceIdsSchema,
  outcomesSchema,
  passedSchema,
  resourceIdsSchema,
  severitiesSchema,
  statusClassesSchema,
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
        namespaceIds: resourceIdsSchema.optional(),
        identifiers: identifiersSchema.optional(),
        passed: passedSchema.optional(),
        eventTypes: eventTypesSchema.optional(),
        outcomes: outcomesSchema.optional(),
        keySpaceIds: keySpaceIdsSchema.optional(),
        statusClasses: statusClassesSchema.optional(),
        severities: severitiesSchema.optional(),
        projectIds: resourceIdsSchema.optional(),
        appIds: resourceIdsSchema.optional(),
        environmentIds: resourceIdsSchema.optional(),
        destination: updateDestinationSchema.optional(),
      })
      .refine(
        (input) =>
          input.name !== undefined ||
          input.status !== undefined ||
          input.namespaceIds !== undefined ||
          input.identifiers !== undefined ||
          input.passed !== undefined ||
          input.eventTypes !== undefined ||
          input.outcomes !== undefined ||
          input.keySpaceIds !== undefined ||
          input.statusClasses !== undefined ||
          input.severities !== undefined ||
          input.projectIds !== undefined ||
          input.appIds !== undefined ||
          input.environmentIds !== undefined ||
          input.destination !== undefined,
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
              await vault.encrypt({
                keyring: ctx.workspace.id,
                data: destination.config.token,
              })
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
          throw new TRPCError({
            code: "NOT_FOUND",
            message: "Log drain not found",
          });
        }
        const existing = decodeLogdrainConfig(drain.config);
        if (
          (existing.stream.kind !== "ratelimits" &&
            (input.namespaceIds !== undefined ||
              input.identifiers !== undefined ||
              input.passed !== undefined)) ||
          (existing.stream.kind !== "key_verifications" &&
            (input.outcomes !== undefined || input.keySpaceIds !== undefined)) ||
          (existing.stream.kind !== "audit_logs" && input.eventTypes !== undefined) ||
          (existing.stream.kind !== "gateway_requests" && input.statusClasses !== undefined) ||
          (existing.stream.kind !== "runtime_logs" && input.severities !== undefined) ||
          (existing.stream.kind !== "gateway_requests" &&
            existing.stream.kind !== "runtime_logs" &&
            (input.projectIds !== undefined ||
              input.appIds !== undefined ||
              input.environmentIds !== undefined))
        ) {
          throw new TRPCError({
            code: "BAD_REQUEST",
            message: "Filters must match the drain stream.",
          });
        }
        let stream: LogdrainConfig["stream"];
        switch (existing.stream.kind) {
          case "ratelimits":
            stream = {
              ...existing.stream,
              namespaceIds: input.namespaceIds ?? existing.stream.namespaceIds,
              identifiers: input.identifiers ?? existing.stream.identifiers,
              passed: input.passed ?? existing.stream.passed,
            };
            break;
          case "runtime_logs":
            stream = {
              ...existing.stream,
              severities: input.severities ?? existing.stream.severities,
              projectIds: input.projectIds ?? existing.stream.projectIds,
              appIds: input.appIds ?? existing.stream.appIds,
              environmentIds: input.environmentIds ?? existing.stream.environmentIds,
            };
            break;
          case "audit_logs":
            stream = {
              ...existing.stream,
              eventTypes: input.eventTypes ?? existing.stream.eventTypes,
            };
            break;
          case "gateway_requests":
            stream = {
              ...existing.stream,
              statusClasses: input.statusClasses ?? existing.stream.statusClasses,
              projectIds: input.projectIds ?? existing.stream.projectIds,
              appIds: input.appIds ?? existing.stream.appIds,
              environmentIds: input.environmentIds ?? existing.stream.environmentIds,
            };
            break;
          case "key_verifications":
            stream = {
              ...existing.stream,
              outcomes: input.outcomes ?? existing.stream.outcomes,
              keySpaceIds: input.keySpaceIds ?? existing.stream.keySpaceIds,
            };
            break;
          default:
            throw new Error(`Unsupported log drain stream: ${existing.stream satisfies never}`);
        }
        let config =
          input.namespaceIds === undefined &&
          input.identifiers === undefined &&
          input.passed === undefined &&
          input.eventTypes === undefined &&
          input.outcomes === undefined &&
          input.statusClasses === undefined &&
          input.severities === undefined &&
          input.projectIds === undefined &&
          input.appIds === undefined &&
          input.environmentIds === undefined &&
          input.keySpaceIds === undefined
            ? drain.config
            : encodeLogdrainConfig({
                ...existing,
                stream,
              });
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
                  stream,
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
                  stream,
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

        const changesDelivery =
          destination !== undefined ||
          input.namespaceIds !== undefined ||
          input.identifiers !== undefined ||
          input.passed !== undefined ||
          input.eventTypes !== undefined ||
          input.outcomes !== undefined ||
          input.statusClasses !== undefined ||
          input.severities !== undefined ||
          input.projectIds !== undefined ||
          input.appIds !== undefined ||
          input.environmentIds !== undefined ||
          input.keySpaceIds !== undefined;
        const resetFailureState = input.status === "running" || changesDelivery;
        const expireLease = input.status !== undefined || changesDelivery;
        const status =
          input.status ??
          (changesDelivery && drain.status === "paused_by_failure" ? "running" : undefined);
        await tx
          .update(schema.logdrains)
          .set({
            ...(input.name !== undefined ? { name: input.name } : {}),
            ...(status !== undefined ? { status } : {}),
            ...(changesDelivery ? { config } : {}),
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
          context: {
            location: ctx.audit.location,
            userAgent: ctx.audit.userAgent,
          },
        });
      });
      return { id: input.id };
    } catch (error) {
      if (error instanceof TRPCError) {
        throw error;
      }
      console.error("Failed to update log drain", error);
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to update log drain",
      });
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
        return {
          ...header,
          name: existingByName.get(normalizedName)?.name ?? header.name,
        };
      }
    }
  });
}

import { VaultService } from "@/gen/proto/vault/v1/service_pb";
import { insertAuditLogs } from "@/lib/audit";
import { db, eq, schema } from "@/lib/db";
import { createVaultClient } from "@/lib/vault-client";
import { TRPCError } from "@trpc/server";
import { newId } from "@unkey/id";
import { z } from "zod";
import { workspaceProcedure } from "../../trpc";
import { type LogdrainConfig, encodeLogdrainConfig, encryptHttpHeaders } from "./config";
import {
  eventTypesSchema,
  httpFormatSchema,
  httpHeadersSchema,
  httpsUrl,
  keySpaceIdsSchema,
  outcomesSchema,
  passedSchema,
  resourceIdsSchema,
  severitiesSchema,
  statusClassesSchema,
} from "./validation";

const vault = createVaultClient(VaultService);
const streamSchema = z.enum([
  "audit_logs",
  "key_verifications",
  "gateway_requests",
  "runtime_logs",
  "ratelimits",
]);

const destinationSchema = z.discriminatedUnion("kind", [
  z.object({
    kind: z.literal("http"),
    config: z.object({
      url: httpsUrl,
      format: httpFormatSchema.default("json"),
      headers: httpHeadersSchema.optional(),
    }),
  }),
  z.object({
    kind: z.literal("axiom"),
    config: z.object({
      dataset: z.string().min(1),
      token: z.string().min(1),
    }),
  }),
]);

export const createLogdrain = workspaceProcedure
  .input(
    z
      .object({
        name: z.string().trim().min(1).max(128),
        batchSize: z.number().int().min(1).max(4_294_967_295).optional(),
        stream: streamSchema.default("audit_logs"),
        namespaceIds: resourceIdsSchema.optional(),
        passed: passedSchema.optional(),
        eventTypes: eventTypesSchema.optional(),
        outcomes: outcomesSchema.optional(),
        keySpaceIds: keySpaceIdsSchema.optional(),
        statusClasses: statusClassesSchema.optional(),
        severities: severitiesSchema.optional(),
        projectIds: resourceIdsSchema.optional(),
        appIds: resourceIdsSchema.optional(),
        environmentIds: resourceIdsSchema.optional(),
      })
      .refine(
        (input) =>
          (input.stream === "ratelimits" ||
            (input.namespaceIds === undefined && input.passed === undefined)) &&
          (input.stream === "key_verifications" ||
            (input.outcomes === undefined && input.keySpaceIds === undefined)) &&
          (input.stream === "audit_logs" || input.eventTypes === undefined) &&
          (input.stream === "gateway_requests" || input.statusClasses === undefined) &&
          (input.stream === "runtime_logs" || input.severities === undefined) &&
          (input.stream === "gateway_requests" ||
            input.stream === "runtime_logs" ||
            (input.projectIds === undefined &&
              input.appIds === undefined &&
              input.environmentIds === undefined)),
        "Filters must match the drain stream.",
      )
      .and(destinationSchema),
  )
  .mutation(async ({ ctx, input }) => {
    const id = newId("logdrain");

    try {
      let stream: LogdrainConfig["stream"];
      switch (input.stream) {
        case "ratelimits":
          stream = {
            kind: input.stream,
            namespaceIds: input.namespaceIds ?? [],
            passed: input.passed ?? [],
          };
          break;
        case "audit_logs":
          stream = { kind: input.stream, eventTypes: input.eventTypes ?? [] };
          break;
        case "key_verifications":
          stream = {
            kind: input.stream,
            outcomes: input.outcomes ?? [],
            keySpaceIds: input.keySpaceIds ?? [],
          };
          break;
        case "gateway_requests":
          stream = {
            kind: input.stream,
            statusClasses: input.statusClasses ?? [],
            projectIds: input.projectIds ?? [],
            appIds: input.appIds ?? [],
            environmentIds: input.environmentIds ?? [],
          };
          break;
        case "runtime_logs":
          stream = {
            kind: input.stream,
            severities: input.severities ?? [],
            projectIds: input.projectIds ?? [],
            appIds: input.appIds ?? [],
            environmentIds: input.environmentIds ?? [],
          };
          break;
        default:
          throw new Error(`Unsupported log drain stream: ${input.stream satisfies never}`);
      }
      let config: Buffer;
      switch (input.kind) {
        case "http":
          config = encodeLogdrainConfig({
            kind: input.kind,
            stream,
            batchSize: input.batchSize,
            url: input.config.url,
            format: input.config.format,
            headers: await encryptHttpHeaders(ctx.workspace.id, input.config.headers ?? {}),
          });
          break;
        case "axiom":
          config = encodeLogdrainConfig({
            kind: input.kind,
            stream,
            batchSize: input.batchSize,
            dataset: input.config.dataset,
            encryptedToken: (
              await vault.encrypt({
                keyring: ctx.workspace.id,
                data: input.config.token,
              })
            ).encrypted,
          });
          break;
        default:
          throw new Error(`Unsupported log drain sink: ${input satisfies never}`);
      }
      const now = Date.now();

      await db.transaction(async (tx) => {
        const [limits] = await tx
          .select({ logdrainsMax: schema.limits.logdrainsMax })
          .from(schema.limits)
          .where(eq(schema.limits.workspaceId, ctx.workspace.id))
          .for("update");
        if (!limits || limits.logdrainsMax === 0) {
          throw new TRPCError({
            code: "FORBIDDEN",
            message: "Contact support to enable log drains for this workspace.",
          });
        }

        const drains = await tx
          .select({ id: schema.logdrains.id })
          .from(schema.logdrains)
          .where(eq(schema.logdrains.workspaceId, ctx.workspace.id))
          .for("update");
        if (drains.length >= limits.logdrainsMax) {
          throw new TRPCError({
            code: "FORBIDDEN",
            message:
              "Log drain limit reached. Contact support to increase this workspace's allowance.",
          });
        }

        await tx.insert(schema.logdrains).values({
          id,
          workspaceId: ctx.workspace.id,
          name: input.name,
          stream: input.stream,
          config,
          committedOffsetInsertedAt: now,
          leaseId: "",
          fencingToken: "",
          createdAt: now,
          updatedAt: now,
        });
        await insertAuditLogs(tx, {
          workspaceId: ctx.workspace.id,
          actor: { type: "user", id: ctx.user.id },
          event: "logdrain.create",
          description: `Created log drain ${id}`,
          resources: [{ type: "logdrain", id, name: input.name }],
          context: {
            location: ctx.audit.location,
            userAgent: ctx.audit.userAgent,
          },
        });
      });

      return { id };
    } catch (error) {
      if (error instanceof TRPCError) {
        throw error;
      }
      console.error("Failed to create log drain", error);
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to create log drain",
      });
    }
  });

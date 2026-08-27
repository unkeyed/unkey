import { VaultService } from "@/gen/proto/vault/v1/service_pb";
import { insertAuditLogs } from "@/lib/audit";
import { and, db, eq, schema } from "@/lib/db";
import { createVaultClient } from "@/lib/vault-client";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { workspaceProcedure } from "../../trpc";
import { decodeLogdrainConfig, encodeLogdrainConfig, encryptHttpHeaders } from "./config";
import { httpFormatSchema, httpHeadersSchema, httpsUrl } from "./validation";

const vault = createVaultClient(VaultService);
const updateDestinationSchema = z.discriminatedUnion("kind", [
  z.object({
    kind: z.literal("http"),
    config: z.object({
      url: httpsUrl,
      format: httpFormatSchema.optional(),
      headers: httpHeadersSchema.optional(),
    }),
  }),
  z.object({
    kind: z.literal("axiom"),
    // url overrides the Axiom base domain for edge deployments; defaults to
    // https://api.axiom.co in the delivery service when absent.
    config: z.object({
      dataset: z.string().min(1),
      token: z.string().min(1).optional(),
      url: httpsUrl.optional(),
    }),
  }),
]);

export const updateLogdrain = workspaceProcedure
  .input(
    z
      .object({
        id: z.string().min(1),
        name: z.string().trim().min(1).max(128).optional(),
        status: z.enum(["enabled", "disabled"]).optional(),
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
      const encryptedHeaders =
        destination?.kind === "http" && destination.config.headers !== undefined
          ? await encryptHttpHeaders(ctx.workspace.id, destination.config.headers)
          : undefined;
      const encryptedToken =
        destination?.kind === "axiom" && destination.config.token !== undefined
          ? (await vault.encrypt({ keyring: ctx.workspace.id, data: destination.config.token }))
              .encrypted
          : undefined;
      await db.transaction(async (tx) => {
        const [drain] = await tx
          .select({
            id: schema.logdrains.id,
            config: schema.logdrains.config,
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
        if (destination && destination.kind !== existing.kind) {
          throw new TRPCError({
            code: "BAD_REQUEST",
            message: "Kind cannot be changed. Create a new log drain instead.",
          });
        }
        let config: Buffer | undefined;
        if (destination?.kind === "http" && existing.kind === "http") {
          config = encodeLogdrainConfig({
            kind: destination.kind,
            url: destination.config.url,
            format: destination.config.format ?? existing.format,
            headers: encryptedHeaders ?? existing.headers,
          });
        } else if (destination?.kind === "axiom" && existing.kind === "axiom") {
          config = encodeLogdrainConfig({
            kind: destination.kind,
            dataset: destination.config.dataset,
            url: destination.config.url ?? existing.url,
            encryptedToken: encryptedToken ?? existing.encryptedToken,
          });
        }

        await tx
          .update(schema.logdrains)
          .set({
            ...(input.name !== undefined ? { name: input.name } : {}),
            ...(input.status !== undefined ? { enabled: input.status === "enabled" } : {}),
            ...(destination
              ? {
                  config,
                }
              : {}),
          })
          .where(
            and(
              eq(schema.logdrains.id, input.id),
              eq(schema.logdrains.workspaceId, ctx.workspace.id),
            ),
          );
        if (input.status !== undefined || destination !== undefined) {
          const resetFailureState = input.status === "enabled" || destination !== undefined;
          await tx
            .update(schema.logdrainState)
            .set({
              // Expire the current lease so a status or destination change
              // fences in-flight state writes and requires a new fencing token.
              leaseExpiresAt: 0,
              ...(resetFailureState
                ? {
                    status: "active" as const,
                    consecutiveFailures: 0,
                    nextAttemptAt: 0,
                    lastError: null,
                  }
                : {}),
            })
            .where(eq(schema.logdrainState.logdrainId, input.id));
        }
        await insertAuditLogs(tx, {
          workspaceId: ctx.workspace.id,
          actor: { type: "user", id: ctx.user.id },
          event: "logdrain.update",
          description: `Updated log drain ${input.id}`,
          resources: [],
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

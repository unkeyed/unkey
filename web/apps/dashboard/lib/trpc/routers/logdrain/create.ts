import { VaultService } from "@/gen/proto/vault/v1/service_pb";
import { insertAuditLogs } from "@/lib/audit";
import { db, schema } from "@/lib/db";
import { createVaultClient } from "@/lib/vault-client";
import { TRPCError } from "@trpc/server";
import { nanoid } from "nanoid";
import { z } from "zod";
import { workspaceProcedure } from "../../trpc";
import { encodeLogdrainConfig, encryptHttpHeaders } from "./config";
import { httpFormatSchema, httpHeadersSchema, httpsUrl } from "./validation";

const vault = createVaultClient(VaultService);
const streamSchema = z.enum(["audit_logs"]);

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
    // url overrides the Axiom base domain for edge deployments; defaults to
    // https://api.axiom.co in the delivery service when absent.
    config: z.object({
      dataset: z.string().min(1),
      token: z.string().min(1),
      url: httpsUrl.optional(),
    }),
  }),
]);

export const createLogdrain = workspaceProcedure
  .input(
    z
      .object({
        name: z.string().trim().min(1).max(128),
        stream: streamSchema.default("audit_logs"),
        startFrom: z.enum(["now", "beginning"]).default("now"),
      })
      .and(destinationSchema),
  )
  .mutation(async ({ ctx, input }) => {
    const id = `ld_${nanoid(12)}`;

    try {
      const config =
        input.kind === "http"
          ? encodeLogdrainConfig({
              kind: input.kind,
              url: input.config.url,
              format: input.config.format,
              headers: await encryptHttpHeaders(ctx.workspace.id, input.config.headers ?? {}),
            })
          : encodeLogdrainConfig({
              kind: input.kind,
              dataset: input.config.dataset,
              url: input.config.url ?? "",
              encryptedToken: (
                await vault.encrypt({ keyring: ctx.workspace.id, data: input.config.token })
              ).encrypted,
            });
      const now = Date.now();
      const initialOffset = input.startFrom === "beginning" ? 0 : now;

      await db.transaction(async (tx) => {
        await tx.insert(schema.logdrains).values({
          id,
          workspaceId: ctx.workspace.id,
          name: input.name,
          stream: input.stream,
          config,
          createdAt: now,
          updatedAt: now,
        });
        await tx.insert(schema.logdrainState).values({
          logdrainId: id,
          committedOffsetInsertedAt: initialOffset,
        });
        await insertAuditLogs(tx, {
          workspaceId: ctx.workspace.id,
          actor: { type: "user", id: ctx.user.id },
          event: "logdrain.create",
          description: `Created log drain ${id}`,
          resources: [],
          context: { location: ctx.audit.location, userAgent: ctx.audit.userAgent },
        });
      });

      return { id };
    } catch (error) {
      console.error("Failed to create log drain", error);
      throw new TRPCError({ code: "INTERNAL_SERVER_ERROR", message: "Failed to create log drain" });
    }
  });

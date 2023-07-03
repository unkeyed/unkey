import { db, schema, eq } from "@unkey/db";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { t, auth } from "../trpc";
import { Kafka } from "@upstash/kafka";
import { env } from "@/lib/env";

import { unkeyRoot, unkeyScoped } from "@/lib/api";
const kafka = new Kafka({
  url: env.UPSTASH_KAFKA_REST_URL,
  username: env.UPSTASH_KAFKA_REST_USERNAME,
  password: env.UPSTASH_KAFKA_REST_PASSWORD,
});

export const keyRouter = t.router({
  create: t.procedure
    .use(auth)
    .input(
      z.object({
        prefix: z.string().optional(),
        bytesLength: z.number().int().gte(1).default(16),
        apiId: z.string(),
        ownerId: z.string().nullish(),
        meta: z.record(z.unknown()).optional(),
        expires: z.number().int().nullish(), // unix timestamp in milliseconds

        ratelimit: z
          .object({
            type: z.enum(["consistent", "fast"]),
            refillInterval: z.number().int().positive(),
            refillRate: z.number().int().positive(),
            limit: z.number().int().positive(),
          })
          .optional(),
      }),
    )
    .mutation(async ({ input, ctx }) => {
      const workspace = await db.query.workspaces.findFirst({
        where: eq(schema.workspaces.tenantId, ctx.tenant.id),
      });
      if (!workspace) {
        throw new TRPCError({ code: "NOT_FOUND", message: "workspace not found" });
      }

      const newRootKey = await unkeyRoot._internal.createRootKey({
        forWorkspaceId: workspace.id,
        name: "Dashboard",
        expires: Date.now() + 60000, // expires in 1 minute
      });

      return unkeyScoped(newRootKey.key).keys.create({
        apiId: input.apiId,
        prefix: input.prefix,
        byteLength: input.bytesLength,
        ownerId: input.ownerId ?? undefined,
        meta: input.meta,
        expires: input.expires ?? undefined,
        ratelimit: input.ratelimit,
      });
    }),
  createInternalRootKey: t.procedure.use(auth).mutation(async ({ ctx }) => {
    const unkeyApi = await db.query.apis.findFirst({
      where: eq(schema.apis.id, env.UNKEY_API_ID),
      with: {
        workspace: true,
      },
    });
    if (!unkeyApi) {
      console.error(`api ${env.UNKEY_API_ID} not found`);
      throw new TRPCError({ code: "NOT_FOUND", message: `api ${env.UNKEY_API_ID} not found` });
    }

    const workspace = await db.query.workspaces.findFirst({
      where: eq(schema.workspaces.tenantId, ctx.tenant.id),
    });
    if (!workspace) {
      console.error(`workspace for tenant ${ctx.tenant.id} not found`);
      throw new TRPCError({ code: "NOT_FOUND", message: "workspace not found" });
    }
    console.log({ workspace });

    const newRootKey = await unkeyRoot._internal
      .createRootKey({
        forWorkspaceId: workspace.id,
      })
      .catch((err) => {
        console.error(err);
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message: `unable to create root key: ${err.message}`,
        });
      });

    return newRootKey;
  }),
  delete: t.procedure
    .use(auth)
    .input(
      z.object({
        keyIds: z.array(z.string()),
      }),
    )
    .mutation(async ({ ctx, input }) => {
      await Promise.all(
        input.keyIds.map(async (keyId) => {
          const where = eq(schema.keys.id, keyId);

          const key = await db.query.keys.findFirst({
            where,
          });
          console.log({ keyId, key }, ctx.tenant);

          if (!key) {
            throw new TRPCError({ code: "NOT_FOUND", message: "key not found" });
          }

          const workspace = await db.query.workspaces.findFirst({
            where: eq(schema.workspaces.id, key.forWorkspaceId ?? key.workspaceId),
          });
          if (!workspace) {
            throw new TRPCError({ code: "NOT_FOUND", message: "workspace not found" });
          }
          if (workspace.tenantId !== ctx.tenant.id) {
            throw new TRPCError({ code: "UNAUTHORIZED" });
          }

          await db.delete(schema.keys).where(where);
          await kafka.producer().produce("key.changed", {
            type: "deleted",
            key: {
              id: key.id,
              hash: key.hash,
            },
          });
        }),
      );
      return;
    }),
});

import { db, schema, eq, Key } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { t, auth } from "../trpc";
import { newId } from "@unkey/id";
import { toBase58 } from "@/lib/api/base58";
import { toBase64 } from "@/lib/api/base64";
import { Policy, type GRID } from "@unkey/policies";
import { Kafka } from "@upstash/kafka";
import { env } from "@/lib/env";

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
        bytes: z.number().int().gte(1).default(16),
        apiId: z.string(),
        ownerId: z.string().nullish(),
        meta: z.record(z.unknown()).optional(),
        expires: z.number().int().nullish(), // unix timestamp in milliseconds
        limit: z.number().int().positive().nullish(),
        ratelimit: z
          .object({
            type: z.enum(["consistent", "fast"]),
            refillInterval: z.number().int().positive(),
            refillRate: z.number().int().positive(),
            limit: z.number().int().positive(),
          })
          .nullish(),
      }),
    )
    .mutation(async ({ input, ctx }) => {
      const workspace = await db.query.workspaces.findFirst({
        where: eq(schema.workspaces.tenantId, ctx.tenant.id),
      });
      if (!workspace) {
        throw new TRPCError({ code: "NOT_FOUND", message: "workspace not found" });
      }
      const buf = new Uint8Array(input.bytes);
      crypto.getRandomValues(buf);

      let key = toBase58(buf);
      if (input.prefix) {
        key = [input.prefix, key].join("_");
      }

      const hash = toBase64(await crypto.subtle.digest("sha-256", new TextEncoder().encode(key)));

      const id = newId("key");

      const values: Key = {
        id,
        apiId: input.apiId,
        hash,
        ownerId: input.ownerId ?? null,
        meta: input.meta,
        start: key.substring(0, key.indexOf("_") + 4),
        createdAt: new Date(),
        expires: input.expires ? new Date(input.expires) : null,
        // remaining: input.limit ?? null,
        // remainingRequests: input.limit ?? null,
        ratelimitType: null,
        ratelimitRefillInterval: null,
        ratelimitRefillRate: null,
        ratelimitLimit: null,
        workspaceId: workspace.id,
        forWorkspaceId: null,
      };

      if (input.ratelimit) {
        values.ratelimitType = input.ratelimit.type;
        values.ratelimitRefillInterval = input.ratelimit.refillInterval;
        values.ratelimitRefillRate = input.ratelimit.refillRate;
        values.ratelimitLimit = input.ratelimit.limit;
      }

      await db.insert(schema.keys).values(values).execute();
      const policyId = newId("policy");
      await db
        .insert(schema.policies)
        .values({
          id: policyId,
          createdAt: new Date(),
          name: "root",
          policy: new Policy([
            {
              resources: {
                api: {
                  [`${workspace.id}::api::*` satisfies GRID]: [
                    "create",
                    "read",
                    "update",
                    "delete",
                    "create:key",
                  ],
                },
                key: {
                  [`${workspace.id}::key::*` satisfies GRID]: [
                    "create",
                    "read",
                    "update",
                    "delete",
                    "attach:policy",
                    "detach:policy",
                  ],
                },
                policy: {
                  [`${workspace.id}::policy::*` satisfies GRID]: [
                    "create",
                    "read",
                    "update",
                    "delete",
                  ],
                },
              },
            },
          ]).toString(),
          updatedAt: new Date(),
          version: "v1",
        })
        .execute();

      await db
        .insert(schema.keysToPolicies)
        .values({
          keyId: values.id,
          policyId,
        })
        .execute();
      return {
        key,
        id,
      };
    }),
  createInternalRootKey: t.procedure.use(auth).mutation(async ({ ctx, input }) => {
    const buf = new Uint8Array(16);
    crypto.getRandomValues(buf);

    const key = ["unkey", toBase58(buf)].join("_");

    const hash = toBase64(await crypto.subtle.digest("sha-256", new TextEncoder().encode(key)));

    const id = newId("key");

    const api = await db.query.apis.findFirst({
      where: eq(schema.apis.id, env.UNKEY_API_ID),
      with: {
        workspace: true,
      },
    });
    if (!api) {
      throw new TRPCError({ code: "NOT_FOUND", message: `api ${env.UNKEY_API_ID} not found` });
    }

    const workspace = await db.query.workspaces.findFirst({
      where: eq(schema.workspaces.tenantId, ctx.tenant.id),
    });
    if (!workspace) {
      throw new TRPCError({ code: "NOT_FOUND", message: "workspace not found" });
    }
    await db
      .insert(schema.keys)
      .values({
        id,
        hash,
        apiId: env.UNKEY_API_ID,
        start: key.substring(0, key.indexOf("_") + 4),
        createdAt: new Date(),
        workspaceId: env.UNKEY_WORKSPACE_ID,
        forWorkspaceId: workspace.id,
      })
      .execute();

    return {
      key,
      id,
    };
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
          await kafka.producer().produce("key.deleted", {
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

import { db, schema, eq, Key } from "@unkey/db";
import { TRPCError } from "@trpc/server";
import { z } from "zod";

import { t, auth } from "../trpc";
import { newId } from "@unkey/id";
import { toBase58 } from "@/lib/api/base58";
import { toBase64 } from "@/lib/api/base64";
import { Policy, type GRID } from "@unkey/policies";

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
        expires: z.number().int().optional(), // unix timestamp in milliseconds
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
        workspaceId: ctx.workspace.id,
        hash,
        ownerId: input.ownerId ?? null,
        meta: input.meta,
        start: key.substring(0, key.indexOf("_") + 4),
        createdAt: new Date(),
        internal: false,
        expires: input.expires ? new Date(input.expires) : null,
        ratelimitType: null,
        ratelimitRefillInterval: null,
        ratelimitRefillRate: null,
        ratelimitLimit: null,
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
                  [`${ctx.workspace.id}::api::*` satisfies GRID]: [
                    "create",
                    "read",
                    "update",
                    "delete",
                    "create:key",
                  ],
                },
                key: {
                  [`${ctx.workspace.id}::key::*` satisfies GRID]: [
                    "create",
                    "read",
                    "update",
                    "delete",
                    "attach:policy",
                    "detach:policy",
                  ],
                },
                policy: {
                  [`${ctx.workspace.id}::policy::*` satisfies GRID]: [
                    "create",
                    "read",
                    "update",
                    "delete",
                  ],
                },
              },
            },
          ]).toString(),
          workspaceId: ctx.workspace.id,
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
  createInternalRootKey: t.procedure.use(auth)

  .mutation(async ({ ctx }) => {
    const buf = new Uint8Array(16);
    crypto.getRandomValues(buf);

    const key = ["unkey", toBase58(buf)].join("_");

    const hash = toBase64(await crypto.subtle.digest("sha-256", new TextEncoder().encode(key)));

    const id = newId("key");

    await db
      .insert(schema.keys)
      .values({
        id,
        workspaceId: ctx.workspace.id,
        hash,
        start: key.substring(0, key.indexOf("_") + 4),
        createdAt: new Date(),
        internal: true,
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
        keyId: z.string(),
      }),
    )
    .mutation(async ({ ctx, input }) => {
      const where = eq(schema.keys.id, input.keyId);

      const key = await db.query.keys.findFirst({
        where,
      });

      if (!key || key.workspaceId !== ctx.workspace.id) {
        throw new TRPCError({ code: "NOT_FOUND", message: "key not found" });
      }

      await db.delete(schema.keys).where(where);

      return;
    }),
});

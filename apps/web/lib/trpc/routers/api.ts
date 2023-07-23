import { db, schema, type Key } from "@unkey/db";
import { TRPCError } from "@trpc/server";
import { z } from "zod";

import { t, auth } from "../trpc";
import { newId } from "@unkey/id";
import { eq } from "drizzle-orm";
import { Kafka } from "@upstash/kafka";
import { env } from "@/lib/env";

const kafka = new Kafka({
  url: env.UPSTASH_KAFKA_REST_URL,
  username: env.UPSTASH_KAFKA_REST_USERNAME,
  password: env.UPSTASH_KAFKA_REST_PASSWORD,
});
const producer = kafka.producer();

export const apiRouter = t.router({
  delete: t.procedure
    .use(auth)
    .input(
      z.object({
        apiId: z.string(),
      }),
    )
    .mutation(async ({ ctx, input }) => {
      const api = await db.query.apis.findFirst({
        where: eq(schema.apis.id, input.apiId),
        with: {
          workspace: true,
        },
      });
      // Check if the API exists and if the user owns it
      if (!api || api.workspace?.tenantId !== ctx.tenant.id) {
        throw new TRPCError({ code: "NOT_FOUND", message: "api not found" });
      }

      // delete keys for the api
      let keys: Key[] = [];
      do {
        keys = await db.query.keys.findMany({
          where: eq(schema.keys.apiId, input.apiId),
        });
        await Promise.all(
          keys.map(async (key) => {
            await db.delete(schema.keys).where(eq(schema.keys.id, key.id));
            await producer.produce("key.deleted", {
              key: {
                id: key.id,
                hash: key.hash,
              },
            });
          }),
        );
      } while (keys.length > 0);

      // delete api
      await db.delete(schema.apis).where(eq(schema.apis.id, input.apiId));
      return;
    }),
  create: t.procedure
    .use(auth)
    .input(
      z.object({
        name: z.string().min(1).max(50),
      }),
    )
    .mutation(async ({ input, ctx }) => {
      const id = newId("api");
      const workspace = await db.query.workspaces.findFirst({
        where: eq(schema.workspaces.tenantId, ctx.tenant.id),
      });
      if (!workspace) {
        throw new TRPCError({ code: "NOT_FOUND", message: "workspace not found" });
      }

      const keyAuth = {
        id: newId("keyAuth"),
        workspaceId: workspace.id,
      };
      await db.insert(schema.keyAuth).values(keyAuth);

      await db.insert(schema.apis).values({
        id,
        workspaceId: workspace.id,
        name: input.name,
        authType: "key",
        keyAuthId: keyAuth.id,
      });

      return {
        id,
      };
    }),
});

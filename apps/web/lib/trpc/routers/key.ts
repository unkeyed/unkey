import { db, schema } from "@unkey/db";
import { TRPCError } from "@trpc/server";
import { z } from "zod";

import { t, auth } from "../trpc";
import { newId } from "@unkey/id";
import { toBase58 } from "@/lib/api/base58";
import { toBase64 } from "@/lib/api/base64";

export const keyRouter = t.router({
  // list: t.procedure.use(auth).query(async ({ ctx }) => {
  //   return await db.channel.findMany({
  //     where: {
  //       tenantId: ctx.tenant.id,
  //     },
  //   });
  // }),
  // delete: t.procedure
  //   .use(auth)
  //   .input(
  //     z.object({
  //       channelId: z.string(),
  //     }),
  //   )
  //   .mutation(async ({ input, ctx }) => {
  //     const channel = await db.channel.findFirst({
  //       where: {
  //         AND: {
  //           id: input.channelId,
  //           tenantId: ctx.tenant.id,
  //         },
  //       },
  //     });
  //     if (!channel) {
  //       throw new TRPCError({ code: "NOT_FOUND" });
  //     }
  //     await db.channel.delete({
  //       where: {
  //         id: channel.id,
  //       },
  //     });
  //   }),
  create: t.procedure
    .use(auth)
    .input(
      z.object({
        prefix: z.string().optional(),
        bytes: z.number().int().gte(1).default(16),
        apiId: z.string(),
        ownerId: z.string().optional(),
        meta: z.record(z.unknown()).optional(),
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

      await db
        .insert(schema.keys)
        .values({
          id,
          apiId: input.apiId,
          tenantId: ctx.tenant.id,
          hash,
          ownerId: input.ownerId,
          meta: input.meta,
          start: key.substring(0, (input.prefix?.length ?? 0) + 4),
          createdAt: new Date(),
        })
        .execute();

      return {
        key,
        id,
      };
    }),
});

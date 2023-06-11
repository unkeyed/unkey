import { db, schema } from "@unkey/db";
import { TRPCError } from "@trpc/server";
import { z } from "zod";

import { t, auth } from "../trpc";
import { newId } from "@unkey/id";

export const apiRouter = t.router({
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
        name: z.string().min(1).max(50),
        slug: z.string().min(1).max(50).regex(/^[a-zA-Z0-9-_\.]+$/),
      }),
    )
    .mutation(async ({ input, ctx }) => {
      const id = newId("api");
      await db.insert(schema.apis).values({
        id,
        tenantId: ctx.tenant.id,
        name: input.name,
        slug: input.slug,
      });

      return {
        id,
      };
    }),
});

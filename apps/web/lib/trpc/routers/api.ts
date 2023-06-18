import { db, schema } from "@unkey/db";
import { TRPCError } from "@trpc/server";
import { z } from "zod";

import { t, auth } from "../trpc";
import { newId } from "@unkey/id";
import { eq } from "drizzle-orm";

export const apiRouter = t.router({
  // list: t.procedure.use(auth).query(async ({ ctx }) => {
  //   return await db.channel.findMany({
  //     where: {
  //       workspaceId: ctx.workspace.id,
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
  //           workspaceId: ctx.workspace.id,
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

      await db.insert(schema.apis).values({
        id,
        workspaceId: workspace.id,
        name: input.name,
      });

      return {
        id,
      };
    }),
});

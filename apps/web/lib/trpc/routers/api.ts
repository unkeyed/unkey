import { TRPCError } from "@trpc/server";
import { z } from "zod";

import { unkeyScoped } from "@/lib/api";
import { auth, t } from "../trpc";

export const apiRouter = t.router({
  delete: t.procedure
    .use(auth)
    .input(
      z.object({
        apiId: z.string(),
      }),
    )
    .mutation(async ({ ctx, input }) => {
      if (!ctx.rootKey) {
        throw new TRPCError({ code: "INTERNAL_SERVER_ERROR", message: "unable to load rootKey" });
      }
      const res = await unkeyScoped(ctx.rootKey).apis.remove({ apiId: input.apiId });
      if (res.error) {
        throw new TRPCError({ code: "INTERNAL_SERVER_ERROR", message: res.error.message });
      }
    }),
  create: t.procedure
    .use(auth)
    .input(
      z.object({
        name: z.string().min(1).max(50),
      }),
    )
    .mutation(async ({ input, ctx }) => {
      if (!ctx.rootKey) {
        throw new TRPCError({ code: "INTERNAL_SERVER_ERROR", message: "unable to load rootKey" });
      }
      const res = await unkeyScoped(ctx.rootKey).apis.create({ name: input.name });
      if (res.error) {
        throw new TRPCError({ code: "INTERNAL_SERVER_ERROR", message: res.error.message });
      }

      return {
        id: res.result.apiId,
      };
    }),
});

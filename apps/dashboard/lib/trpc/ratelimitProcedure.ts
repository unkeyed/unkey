import { TRPCError } from "@trpc/server";
import { Ratelimit } from "@unkey/ratelimit";
import { env } from "../env";
// Values for route types
import { auth, protectedProcedure } from "./trpc";

export const ratelimit = env().UNKEY_ROOT_KEY ? {

  create: new Ratelimit({
    rootKey: env().UNKEY_ROOT_KEY ?? "",
    namespace: "trpc_create",
    limit: 5,
    duration: "3s",
  }),

  update: new Ratelimit({
    rootKey: env().UNKEY_ROOT_KEY ?? "",
    namespace: "trpc_update",
    limit: 25,
    duration: "5s",
  }),
  delete: new Ratelimit({
    rootKey: env().UNKEY_ROOT_KEY ?? "",
    namespace: "trpc_delete",
    limit: 5,
    duration: "5s",
  }),
}: {

};
export const rateLimitedProcedure = (ratelimit: Ratelimit | undefined) => ratelimit ?
  protectedProcedure.use(async (opts) => {
   
    const response = await ratelimit.limit(opts.ctx.user.id);

    if (!response.success) {
      throw new TRPCError({
        code: "TOO_MANY_REQUESTS",
        message: "Too many requests in the allowed duration. Please try again",
      });
    }

    return opts.next({
      ctx: {
        ...opts.ctx,
        remaining: response.remaining,
      },
    });
  }) : protectedProcedure;

import { TRPCError, initTRPC } from "@trpc/server";
import superjson from "superjson";

import type { Context } from "./context";
import { Ratelimit } from "@unkey/ratelimit";
import { env } from "../env";

export const t = initTRPC.context<Context>().create({ transformer: superjson });

export const auth = t.middleware(({ next, ctx }) => {
  if (!ctx.user?.id) {
    throw new TRPCError({ code: "UNAUTHORIZED" });
  }

  return next({
    ctx: {
      user: ctx.user,
      tenant: ctx.tenant ?? { id: ctx.user.id, role: "owner" },
    },
  });
});

export const protectedProcedure = t.procedure.use(auth);

export const rateLimitedProcedure = ({
  limit,
  duration,
}: {
  limit: number;
  duration: number;
}) =>
  protectedProcedure.use(async (opts) => {
    const unkey = new Ratelimit({
      rootKey: env().UNKEY_ROOT_KEY,
      namespace: `trpc_${opts.path}`,
      limit: limit ?? 3,
      duration: duration ? `${duration}s` : `${5}s`,
    });

    const ratelimit = await unkey.limit(opts.ctx.user.id);
    console.log("login rate limits", ratelimit) 

    if (!ratelimit.success) {
      throw new TRPCError({
        code: "TOO_MANY_REQUESTS",
        message: JSON.stringify(ratelimit),
      });
    }

    return opts.next({
      ctx: {
        ...opts.ctx,
        remaining: ratelimit.remaining,
      },
    });
  });

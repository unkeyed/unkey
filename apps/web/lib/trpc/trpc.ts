import { TRPCError, initTRPC } from "@trpc/server";
import superjson from "superjson";

import { Context } from "./context";

export const t = initTRPC.context<Context>().create({ transformer: superjson });

export const auth = t.middleware(({ next, ctx }) => {
  if (!ctx.user?.id) {
    throw new TRPCError({ code: "UNAUTHORIZED" });
  }
  return next({
    ctx: {
      user: ctx.user,
      tenant: ctx.tenant ?? { id: ctx.user.id, slug: "home", role: "owner" },
    },
  });
});

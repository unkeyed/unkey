import { TRPCError, initTRPC } from "@trpc/server";
import superjson from "superjson";

import type { Context } from "./context";

export const t = initTRPC.context<Context>().create({ transformer: superjson });

export const auth = t.middleware(({ next, ctx }) => {
  if (!ctx.user?.id) {
    throw new TRPCError({ code: "UNAUTHORIZED" });
  }
  if (!ctx.workspace) {
    throw new TRPCError({ code: "NOT_FOUND", message: "workspace not found in context" });
  }

  return next({
    ctx: {
      workspace: ctx.workspace,
      user: ctx.user,
      tenant: ctx.tenant ?? { id: ctx.user.id, role: "owner" },
    },
  });
});

export const protectedProcedure = t.procedure.use(auth);

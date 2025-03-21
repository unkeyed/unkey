import { env } from "@/lib/env";
import { TRPCError, initTRPC } from "@trpc/server";
import { Ratelimit } from "@unkey/ratelimit";
import superjson from "superjson";
import type { Context } from "./context";

export const t = initTRPC.context<Context>().create({ transformer: superjson });

export const requireUser = t.middleware(({ next, ctx }) => {
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

export const requireWorkspace = t.middleware(({ next, ctx }) => {
  if (!ctx.workspace) {
    throw new TRPCError({ code: "NOT_FOUND", message: "workspace not found in context" });
  }

  return next({
    ctx: {
      workspace: ctx.workspace,
    },
  });
});

export const ratelimit = env().UNKEY_ROOT_KEY
  ? {
      create: new Ratelimit({
        rootKey: env().UNKEY_ROOT_KEY ?? "",
        namespace: "trpc_create",
        limit: 25,
        duration: "3s",
      }),
      read: new Ratelimit({
        rootKey: env().UNKEY_ROOT_KEY ?? "",
        namespace: "trpc_read",
        limit: 100,
        duration: "10s",
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
        limit: 25,
        duration: "5s",
      }),
    }
  : {};

export const withRatelimit = (ratelimit: Ratelimit | undefined) =>
  t.middleware(async ({ next, ctx }) => {
    if (!ratelimit) {
      return next();
    }
    const response = await ratelimit.limit(ctx.user!.id);

    if (!response.success) {
      throw new TRPCError({
        code: "TOO_MANY_REQUESTS",
        message: "Too many requests in the allowed duration. Please try again",
      });
    }

    return next();
  });

// Values for route types
import { protectedProcedure } from "./trpc";
import { Ratelimit } from "@unkey/ratelimit";
import { env } from "../env";
import { TRPCError } from "@trpc/server";

//Values are in seconds currently
export const CREATE_LIMIT_SEC = 5;
export const CREATE_LIMIT_DURATION_SEC = "3s";

export const UPDATE_LIMIT_SEC = 25;
export const UPDATE_LIMIT_DURATION_SEC = "5s";

export const DELETE_LIMIT_SEC = 5;
export const DELETE_LIMIT_DURATION_SEC = "3s";

export const ratelimit = {
  create: new Ratelimit({
    rootKey: env().UNKEY_ROOT_KEY,
    namespace: "trpc_create",
    limit: CREATE_LIMIT_SEC ?? 3,
    duration: CREATE_LIMIT_DURATION_SEC,
  }),

  update: new Ratelimit({
    rootKey: env().UNKEY_ROOT_KEY,
    namespace: "trpc_update",
    limit: UPDATE_LIMIT_SEC,
    duration: UPDATE_LIMIT_DURATION_SEC,
  }),
  delete: new Ratelimit({
    rootKey: env().UNKEY_ROOT_KEY,
    namespace: "trpc_delete",
    limit: DELETE_LIMIT_SEC,
    duration: DELETE_LIMIT_DURATION_SEC,
  }),
};
export const rateLimitedProcedure = (ratelimit: Ratelimit) =>
  protectedProcedure.use(async (opts) => {
    const unkey = ratelimit;

    const response = await unkey.limit(opts.ctx.user.id);

    if (!response.success) {
      throw new TRPCError({
        code: "TOO_MANY_REQUESTS",
        message: JSON.stringify(response),
      });
    }

    return opts.next({
      ctx: {
        ...opts.ctx,
        remaining: response.remaining,
      },
    });
  });

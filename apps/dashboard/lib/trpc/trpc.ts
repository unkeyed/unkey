import { env } from "@/lib/env";
import { TRPCError, initTRPC } from "@trpc/server";
import { Ratelimit } from "@unkey/ratelimit";
import superjson from "superjson";
import { z } from "zod";
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
    throw new TRPCError({
      code: "NOT_FOUND",
      message: "workspace not found in context",
    });
  }

  return next({
    ctx: {
      workspace: ctx.workspace,
    },
  });
});

export const requireSelf = t.middleware(({ next, ctx, rawInput: userId }) => {
  if (ctx.user?.id !== userId) {
    throw new TRPCError({
      code: "FORBIDDEN",
      message: "You can only access your own data",
    });
  }
  return next();
});

export const requireOrgAdmin = t.middleware(async ({ next, ctx, rawInput }) => {
  let orgId: string | undefined;

  // rawInput can be a string with just the orgId, or an object containing the orgId when it passed with other parameters
  if (typeof rawInput === "string") {
    orgId = rawInput;
  } else if (rawInput && typeof rawInput === "object" && "orgId" in rawInput) {
    orgId = rawInput.orgId as string;
  }

  if (!orgId) {
    throw new TRPCError({
      code: "BAD_REQUEST",
      message: "Organization ID is required",
    });
  }
  try {
    const isAdmin = ctx.tenant?.role === "admin";

    if (!isAdmin) {
      throw new TRPCError({
        code: "FORBIDDEN",
        message: "This action requires admin privileges.",
      });
    }

    return next();
  } catch (error) {
    throw new TRPCError({
      code: "INTERNAL_SERVER_ERROR",
      message: "Failed to verify admin privilege.",
      cause: error,
    });
  }
});

const onError = (err: Error, identifier: string) => {
  console.error(`Error occurred while rate limiting ${identifier}: ${err.message}`);
  return { success: true, limit: 0, remaining: 1, reset: 1 };
};

export const ratelimit = env().UNKEY_ROOT_KEY
  ? {
      create: new Ratelimit({
        rootKey: env().UNKEY_ROOT_KEY ?? "",
        namespace: "trpc_create",
        limit: 25,
        duration: "3s",
        onError,
      }),
      read: new Ratelimit({
        rootKey: env().UNKEY_ROOT_KEY ?? "",
        namespace: "trpc_read",
        limit: 100,
        duration: "10s",
        onError,
      }),
      update: new Ratelimit({
        rootKey: env().UNKEY_ROOT_KEY ?? "",
        namespace: "trpc_update",
        limit: 25,
        duration: "5s",
        onError,
      }),
      delete: new Ratelimit({
        rootKey: env().UNKEY_ROOT_KEY ?? "",
        namespace: "trpc_delete",
        limit: 25,
        duration: "5s",
        onError,
      }),
    }
  : {};

export const withRatelimit = (ratelimit: Ratelimit | undefined) =>
  t.middleware(async ({ next, ctx }) => {
    const userId = ctx.user?.id;
    if (!ratelimit || !userId) {
      return next();
    }
    const response = await ratelimit.limit(userId);

    if (!response.success) {
      throw new TRPCError({
        code: "TOO_MANY_REQUESTS",
        message: "Too many requests in the allowed duration. Please try again",
      });
    }

    return next();
  });

export const LLM_LIMITS = {
  MIN_QUERY_LENGTH: 3,
  MAX_QUERY_LENGTH: 120,
  MAX_TOKENS_ESTIMATE: 30, // ~4 chars per token for 120 chars
  RATE_LIMIT: 10,
  RATE_DURATION: "60s",
} as const;

const llmRatelimit = env().UNKEY_ROOT_KEY
  ? new Ratelimit({
      rootKey: env().UNKEY_ROOT_KEY ?? "",
      namespace: "trpc_llm",
      limit: LLM_LIMITS.RATE_LIMIT,
      duration: LLM_LIMITS.RATE_DURATION,
      onError,
    })
  : null;

const llmQuerySchema = z.object({
  query: z
    .string()
    .trim()
    .min(LLM_LIMITS.MIN_QUERY_LENGTH, "Query must be at least 3 characters")
    .max(LLM_LIMITS.MAX_QUERY_LENGTH, "Query cannot exceed 120 characters"),
});

export const withLlmAccess = () =>
  t.middleware(async ({ next, ctx, rawInput }) => {
    const userId = ctx.user?.id;
    if (llmRatelimit && userId) {
      const response = await llmRatelimit.limit(userId);
      if (!response.success) {
        throw new TRPCError({
          code: "TOO_MANY_REQUESTS",
          message: `LLM rate limit exceeded. You can make ${LLM_LIMITS.RATE_LIMIT} requests per minute.`,
        });
      }
    }

    let validatedInput: z.infer<typeof llmQuerySchema>;
    try {
      validatedInput = llmQuerySchema.parse(rawInput);
    } catch (error) {
      if (error instanceof z.ZodError) {
        const firstError = error.errors[0];
        throw new TRPCError({
          code: "BAD_REQUEST",
          message: firstError?.message || "Invalid query format",
        });
      }
      throw new TRPCError({
        code: "BAD_REQUEST",
        message: "Invalid input format",
      });
    }

    return next({
      ctx: {
        validatedQuery: validatedInput.query,
      },
    });
  });

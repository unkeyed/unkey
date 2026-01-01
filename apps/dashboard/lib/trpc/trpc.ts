import { env } from "@/lib/env";
import * as Sentry from "@sentry/nextjs";
import { TRPCError, initTRPC } from "@trpc/server";
import { Ratelimit } from "@unkey/ratelimit";
import superjson from "superjson";
import { z } from "zod";
import type { Context } from "./context";

export const t = initTRPC.context<Context>().create({ transformer: superjson });

/**
 * Sentry middleware for error tracking and performance monitoring
 * Automatically captures errors and attaches RPC input to Sentry events
 */
const sentryMiddleware = t.middleware(
  Sentry.trpcMiddleware({
    attachRpcInput: true,
  }),
);

/**
 * Error filtering middleware to ignore specific errors that we don't want Sentry to track.
 *
 * Currently ignores UNAUTHORIZED, FORBIDDEN, NOT_FOUND, and TOO_MANY_REQUESTS errors.
 */
const errorFilterMiddleware = t.middleware(({ next }) => {
  try {
    return next();
  } catch (error) {
    if (error instanceof TRPCError) {
      // Log expected tRPC errors as breadcrumbs so we can track them but ignore them as errors.
      if (
        error.code === "UNAUTHORIZED" ||
        error.code === "FORBIDDEN" ||
        error.code === "NOT_FOUND" ||
        error.code === "TOO_MANY_REQUESTS"
      ) {
        Sentry.addBreadcrumb({
          category: "tRPC",
          message: `Expected tRPC Error: ${error.message} (${error.code})`,
          level: "info",
        });
        throw error; // Re-throw so the client still receives the proper tRPC error
      }
    }
    throw error; // Re-throw all other errors for Sentry to capture
  }
});

// =============================================================================
// MIDDLEWARE DEFINITIONS
// =============================================================================

/**
 * Middleware: Requires authenticated user
 * Throws UNAUTHORIZED if user is not authenticated
 */
const requireUser = t.middleware(({ next, ctx }) => {
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

/**
 * Middleware: Requires workspace context
 * Throws NOT_FOUND if workspace is not available in context
 */
const requireWorkspace = t.middleware(({ next, ctx }) => {
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

/**
 * Middleware: Requires user to be accessing their own data
 * Throws FORBIDDEN if user ID doesn't match the input
 */
export const requireSelf = t.middleware(({ next, ctx, rawInput: userId }) => {
  if (ctx.user?.id !== userId) {
    throw new TRPCError({
      code: "FORBIDDEN",
      message: "You can only access your own data",
    });
  }
  return next();
});

/**
 * Middleware: Requires organization admin privileges
 * Throws FORBIDDEN if user is not an admin of the organization
 */
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

// =============================================================================
// RATE LIMITING
// =============================================================================

const onError = (err: Error, identifier: string) => {
  console.error(`Error occurred while rate limiting ${identifier}: ${err.message}`);
  return { success: true, limit: 0, remaining: 1, reset: 1 };
};

/**
 * Rate limiters for different operation types
 * Configured with Unkey for distributed rate limiting
 */
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

/**
 * Middleware factory: Adds rate limiting to a procedure
 * @param ratelimit - The Ratelimit instance to use
 * @returns Middleware that enforces rate limits per user
 */
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

/**
 * Rate limits specific to LLM endpoints
 * Stricter limits to prevent abuse of AI-powered features
 */
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

/**
 * Validation schema for LLM queries
 */
const llmQuerySchema = z.object({
  query: z
    .string()
    .trim()
    .min(LLM_LIMITS.MIN_QUERY_LENGTH, "Query must be at least 3 characters")
    .max(LLM_LIMITS.MAX_QUERY_LENGTH, "Query cannot exceed 120 characters"),
});

/**
 * Middleware factory: Adds LLM rate limiting and query validation
 * @returns Middleware that enforces LLM-specific rate limits and validates query format
 */
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

// =============================================================================
// BASE PROCEDURES
// All procedures are built on baseProcedure which includes Sentry middleware
// This ensures every tRPC request is tracked in Sentry
// =============================================================================

/**
 * Base procedure with Sentry middleware
 * All procedures should be built on top of this to ensure Sentry tracking
 */
const baseProcedure = t.procedure.use(errorFilterMiddleware).use(sentryMiddleware);

/**
 * Public procedure - accessible without authentication
 * Includes: Sentry tracking
 *
 * @example
 * export const getPublicData = publicProcedure
 *   .input(z.object({ id: z.string() }))
 *   .query(({ input }) => {
 *     return { data: "public" };
 *   });
 */
export const publicProcedure = baseProcedure;

/**
 * Protected procedure - requires authenticated user
 * Includes: Sentry tracking + user authentication
 *
 * @example
 * export const getUserData = protectedProcedure
 *   .input(z.object({ id: z.string() }))
 *   .query(({ ctx, input }) => {
 *     // ctx.user is guaranteed to exist
 *     return { userId: ctx.user.id };
 *   });
 */
export const protectedProcedure = baseProcedure.use(requireUser);

/**
 * Workspace procedure - requires authenticated user and workspace context
 * Includes: Sentry tracking + user authentication + workspace access
 *
 * @example
 * export const getWorkspaceData = workspaceProcedure
 *   .input(z.object({ id: z.string() }))
 *   .query(({ ctx, input }) => {
 *     // ctx.user and ctx.workspace are guaranteed to exist
 *     return { workspaceId: ctx.workspace.id };
 *   });
 */
export const workspaceProcedure = baseProcedure.use(requireUser).use(requireWorkspace);

// =============================================================================
// UTILITIES
// =============================================================================

/**
 * Router creator
 */
export const router = t.router;

/**
 * Middleware creator
 */
export const middleware = t.middleware;

import { env } from "@/lib/env";
import * as Sentry from "@sentry/nextjs";
import { TRPCError, initTRPC } from "@trpc/server";
import { Ratelimit } from "@unkey/ratelimit";
import superjson from "superjson";
import { z } from "zod";
import {
  generateRequestId,
  logOperation,
  logPerformance,
  logTRPCError,
  logTRPCSuccess,
} from "../logging/structured-logger";
import { classifyError, extractTRPCErrorInfo } from "../utils/error-classification";
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
 * Enhanced error middleware with structured logging
 *
 * Replaces breadcrumb usage with structured logging calls and adds comprehensive
 * request context and timing information for debugging. Expected tRPC errors
 * are logged with structured attributes instead of being reported as Sentry errors.
 *
 * Requirements: 3.1, 3.2, 5.1, 5.2, 5.3
 */
const enhancedErrorMiddleware = t.middleware(({ next, ctx, path, input }) => {
  const startTime = Date.now();
  const requestId = generateRequestId();

  // Create request context for logging
  const requestContext = {
    userId: ctx.user?.id,
    workspaceId: ctx.workspace?.id as string | undefined,
    requestId,
    timestamp: startTime,
  };

  // Log request start with context information
  logOperation(
    "debug",
    "tRPC request started",
    {
      trpc_procedure: path,
      user_id: ctx.user?.id,
      workspace_id: ctx.workspace?.id as string | undefined,
      request_id: requestId,
      has_input: !!input,
      user_agent: ctx.audit?.userAgent,
      location: ctx.audit?.location,
    },
    requestContext,
  );

  try {
    const result = next();

    // Handle successful operations
    if (result instanceof Promise) {
      return result.then(
        (value) => {
          const duration = Date.now() - startTime;

          // Log successful completion with timing
          logTRPCSuccess(path, requestContext, duration);

          // Log performance metrics for slow operations
          if (duration > 1000) {
            logPerformance(`trpc_${path}`, duration, requestContext, {
              slow_operation: 1,
              threshold_ms: 1000,
            });
          }

          return value;
        },
        (error) => {
          const duration = Date.now() - startTime;
          handleTRPCError(error, path, requestContext, duration);
          throw error;
        },
      );
    }
    const duration = Date.now() - startTime;
    logTRPCSuccess(path, requestContext, duration);
    return result;
  } catch (error) {
    const duration = Date.now() - startTime;
    handleTRPCError(error, path, requestContext, duration);
    throw error;
  }
});

/**
 * Handles tRPC errors with enhanced structured logging
 *
 * Classifies errors and logs them appropriately based on whether they are
 * expected (user/validation errors) or unexpected (system errors).
 *
 * @param error - The error that occurred
 * @param path - The tRPC procedure path
 * @param requestContext - Request context for logging
 * @param duration - Request duration in milliseconds
 */
function handleTRPCError(
  error: unknown,
  path: string,
  requestContext: { userId?: string; workspaceId?: string; requestId: string; timestamp: number },
  duration: number,
): void {
  const classification = classifyError(error);

  if (classification.isExpected && classification.errorInfo) {
    // Log expected errors with structured logging instead of breadcrumbs
    logTRPCError(classification.errorInfo, {
      ...requestContext,
      timestamp: requestContext.timestamp, // Keep original timestamp for duration calculation
    });

    // Add additional context for expected errors
    logOperation(
      classification.logLevel,
      "tRPC expected error handled",
      {
        trpc_procedure: path,
        trpc_error_code: classification.errorInfo.code,
        trpc_error_message: classification.errorInfo.message,
        response_time_ms: duration,
        user_id: requestContext.userId,
        workspace_id: requestContext.workspaceId,
        request_id: requestContext.requestId,
        error_classification: "expected",
      },
      requestContext,
    );
  } else {
    // For unexpected errors, log additional context but let Sentry handle the error reporting
    const errorInfo = extractTRPCErrorInfo(error);

    logOperation(
      "error",
      "tRPC unexpected error occurred",
      {
        trpc_procedure: path,
        trpc_error_code: errorInfo?.code || "unknown",
        trpc_error_message:
          errorInfo?.message || (error instanceof Error ? error.message : "Unknown error"),
        response_time_ms: duration,
        user_id: requestContext.userId,
        workspace_id: requestContext.workspaceId,
        request_id: requestContext.requestId,
        error_classification: "unexpected",
        error_type: error instanceof Error ? error.constructor.name : typeof error,
      },
      requestContext,
    );
  }
}

// =============================================================================
// MIDDLEWARE DEFINITIONS
// =============================================================================

/**
 * Middleware: Requires authenticated user with structured logging
 * Throws UNAUTHORIZED if user is not authenticated
 */
const requireUser = t.middleware(({ next, ctx }) => {
  if (!ctx.user?.id) {
    // Log authentication failure
    logOperation("info", "Authentication required but user not found", {
      has_user_object: !!ctx.user,
      user_agent: ctx.audit?.userAgent,
      location: ctx.audit?.location,
    });

    throw new TRPCError({ code: "UNAUTHORIZED" });
  }

  // Log successful authentication
  logOperation("debug", "User authentication verified", {
    user_id: ctx.user.id,
    tenant_id: ctx.tenant?.id || undefined,
    tenant_role: ctx.tenant?.role || undefined,
  });

  return next({
    ctx: {
      user: ctx.user,
      tenant: ctx.tenant ?? { id: ctx.user.id, role: "owner" },
    },
  });
});

/**
 * Middleware: Requires workspace context with structured logging
 * Throws NOT_FOUND if workspace is not available in context
 */
const requireWorkspace = t.middleware(({ next, ctx }) => {
  if (!ctx.workspace) {
    // Log workspace access failure
    logOperation("info", "Workspace required but not found in context", {
      user_id: ctx.user?.id,
      tenant_id: ctx.tenant?.id,
      has_workspace: false,
      user_agent: ctx.audit?.userAgent,
      location: ctx.audit?.location,
    });

    throw new TRPCError({
      code: "NOT_FOUND",
      message: "workspace not found in context",
    });
  }

  // Log successful workspace access
  logOperation("debug", "Workspace access verified", {
    user_id: ctx.user?.id,
    workspace_id: ctx.workspace.id as string,
    workspace_name: ctx.workspace.name as string,
    tenant_id: ctx.tenant?.id || undefined,
  });

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
 * Middleware factory: Adds rate limiting to a procedure with structured logging
 * @param ratelimit - The Ratelimit instance to use
 * @param namespace - The namespace identifier for logging (e.g., "trpc_create", "trpc_read")
 * @returns Middleware that enforces rate limits per user and logs rate limit events
 */
export const withRatelimit = (ratelimit: Ratelimit | undefined, namespace?: string) =>
  t.middleware(async ({ next, ctx }) => {
    const userId = ctx.user?.id;
    if (!ratelimit || !userId) {
      return next();
    }

    const startTime = Date.now();
    const response = await ratelimit.limit(userId);
    const rateLimitDuration = Date.now() - startTime;

    // Log rate limit check with structured logging
    logOperation("debug", "Rate limit check performed", {
      user_id: userId,
      workspace_id: ctx.workspace?.id as string | undefined,
      rate_limit_success: response.success,
      rate_limit_remaining: response.remaining,
      rate_limit_reset: response.reset,
      rate_limit_check_duration_ms: rateLimitDuration,
      rate_limit_namespace: namespace,
    });

    if (!response.success) {
      // Log rate limit exceeded with additional context
      logOperation("warn", "Rate limit exceeded", {
        user_id: userId,
        workspace_id: ctx.workspace?.id as string | undefined,
        rate_limit_remaining: response.remaining,
        rate_limit_reset: response.reset,
        rate_limit_namespace: namespace,
        user_agent: ctx.audit?.userAgent,
        location: ctx.audit?.location,
      });

      throw new TRPCError({
        code: "TOO_MANY_REQUESTS",
        message: "Too many requests in the allowed duration. Please try again",
      });
    }

    return next();
  });

/**
 * Convenience functions for common rate limit patterns with proper namespace logging
 */
export const withCreateRateLimit = () => withRatelimit(ratelimit.create, "trpc_create");
export const withReadRateLimit = () => withRatelimit(ratelimit.read, "trpc_read");
export const withUpdateRateLimit = () => withRatelimit(ratelimit.update, "trpc_update");
export const withDeleteRateLimit = () => withRatelimit(ratelimit.delete, "trpc_delete");

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
 * Middleware factory: Adds LLM rate limiting and query validation with structured logging
 * @returns Middleware that enforces LLM-specific rate limits and validates query format
 */
export const withLlmAccess = () =>
  t.middleware(async ({ next, ctx, rawInput }) => {
    const userId = ctx.user?.id;
    const startTime = Date.now();

    // Log LLM access attempt
    logOperation("debug", "LLM access requested", {
      user_id: userId,
      workspace_id: ctx.workspace?.id as string | undefined,
      has_rate_limit: !!llmRatelimit,
    });

    if (llmRatelimit && userId) {
      const response = await llmRatelimit.limit(userId);
      const rateLimitDuration = Date.now() - startTime;

      // Log LLM rate limit check
      logOperation("debug", "LLM rate limit check performed", {
        user_id: userId,
        workspace_id: ctx.workspace?.id as string | undefined,
        rate_limit_success: response.success,
        rate_limit_remaining: response.remaining,
        rate_limit_reset: response.reset,
        rate_limit_check_duration_ms: rateLimitDuration,
        rate_limit_limit: LLM_LIMITS.RATE_LIMIT,
        rate_limit_duration: LLM_LIMITS.RATE_DURATION,
      });

      if (!response.success) {
        // Log LLM rate limit exceeded
        logOperation("warn", "LLM rate limit exceeded", {
          user_id: userId,
          workspace_id: ctx.workspace?.id as string | undefined,
          rate_limit_remaining: response.remaining,
          rate_limit_reset: response.reset,
          rate_limit_limit: LLM_LIMITS.RATE_LIMIT,
          user_agent: ctx.audit?.userAgent,
          location: ctx.audit?.location,
        });

        throw new TRPCError({
          code: "TOO_MANY_REQUESTS",
          message: `LLM rate limit exceeded. You can make ${LLM_LIMITS.RATE_LIMIT} requests per minute.`,
        });
      }
    }

    let validatedInput: z.infer<typeof llmQuerySchema>;
    try {
      validatedInput = llmQuerySchema.parse(rawInput);

      // Log successful query validation
      logOperation("debug", "LLM query validated", {
        user_id: userId,
        workspace_id: ctx.workspace?.id as string | undefined,
        query_length: validatedInput.query.length,
        query_tokens_estimate: Math.ceil(validatedInput.query.length / 4), // Rough token estimate
      });
    } catch (error) {
      // Log validation failure
      logOperation("warn", "LLM query validation failed", {
        user_id: userId,
        workspace_id: ctx.workspace?.id as string | undefined,
        validation_error:
          error instanceof z.ZodError ? error.issues[0]?.message : "Unknown validation error",
        raw_input_type: typeof rawInput,
      });

      if (error instanceof z.ZodError) {
        const firstError = error.issues[0];
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
 * Base procedure with enhanced error middleware and Sentry tracking
 * All procedures should be built on top of this to ensure comprehensive logging and tracking
 */
const baseProcedure = t.procedure.use(enhancedErrorMiddleware).use(sentryMiddleware);

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

import type { AppRouter } from "@/lib/trpc/routers";
import type { TRPCClientErrorLike } from "@trpc/client";

/**
 * Check if a tRPC error is a WorkOS redirect (307) that should be ignored
 * If a user is genuinely not authed they'd be redirected to the login page.
 */
export const isWorkOSRedirect = (error: TRPCClientErrorLike<AppRouter> | null): boolean => {
  if (!error) {
    return false;
  }

  return error.data?.httpStatus === 307 || error.message?.includes("NEXT_REDIRECT");
};

/**
 * Check if a tRPC error represents an authentication failure
 */
export const isAuthError = (error: TRPCClientErrorLike<AppRouter> | null): boolean => {
  if (!error) {
    return false;
  }

  return error.data?.code === "UNAUTHORIZED" || error.data?.code === "FORBIDDEN";
};

/**
 * Create a retry function for tRPC queries with intelligent error handling
 */
export const createRetryFn =
  (maxRetries = 2) =>
  (failureCount: number, error: TRPCClientErrorLike<AppRouter>) => {
    // Don't retry on auth errors or WorkOS redirects
    if (isAuthError(error) || isWorkOSRedirect(error)) {
      return false;
    }
    return failureCount < maxRetries;
  };

/**
 * Shared query options for consistent tRPC behavior
 */
export const baseQueryOptions = {
  staleTime: 1000 * 60 * 5, // 5 minutes
  gcTime: 1000 * 60 * 15, // 15 minutes (replaces deprecated cacheTime)
  refetchOnWindowFocus: false,
  refetchOnReconnect: true,
};

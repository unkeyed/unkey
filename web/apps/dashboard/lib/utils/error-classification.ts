/**
 * Error classification utilities for tRPC errors
 *
 * This module provides utilities to identify, extract information from, and classify
 * tRPC errors to determine whether they should be filtered from Sentry reporting
 * or logged using structured logging.
 */

/**
 * Expected tRPC error codes that should be filtered from Sentry error reporting
 * These represent normal application flow and user errors, not system issues
 */
export const EXPECTED_TRPC_CODES = [
  "UNAUTHORIZED",
  "FORBIDDEN",
  "NOT_FOUND",
  "TOO_MANY_REQUESTS",
  "BAD_REQUEST", // For validation errors
] as const;

/**
 * Error severity mapping for structured logging
 * Maps tRPC error codes to appropriate log levels
 */
export const ERROR_SEVERITY_MAP = {
  UNAUTHORIZED: "info",
  FORBIDDEN: "info",
  NOT_FOUND: "info",
  TOO_MANY_REQUESTS: "warn",
  BAD_REQUEST: "warn",
  INTERNAL_SERVER_ERROR: "error",
} as const;

/**
 * Interface for extracted tRPC error information
 */
export interface TRPCErrorInfo {
  code: string;
  message: string;
  data?: Record<string, unknown>;
  path?: string;
}

/**
 * Type guard to check if an error code is expected
 */
export function isExpectedErrorCode(code: string): code is (typeof EXPECTED_TRPC_CODES)[number] {
  return (EXPECTED_TRPC_CODES as readonly string[]).includes(code);
}

/**
 * Determines if an error is an expected tRPC error that should be filtered
 *
 * Handles both client-side and server-side tRPC error structures:
 * - Client-side: { data: { code, message }, message }
 * - Server-side: { code, message, data }
 *
 * @param error - The error object to check
 * @returns true if this is an expected tRPC error that should be filtered
 */
export function isExpectedTRPCError(error: unknown): boolean {
  if (!error || typeof error !== "object") {
    return false;
  }

  // Check for tRPC client error structure
  if ("data" in error && error.data && typeof error.data === "object") {
    const data = error.data as Record<string, unknown>;
    if ("code" in data && typeof data.code === "string") {
      return isExpectedErrorCode(data.code);
    }
  }

  // Check for tRPC server error structure
  if ("code" in error && typeof error.code === "string") {
    return isExpectedErrorCode(error.code);
  }

  return false;
}

/**
 * Extracts structured information from a tRPC error
 *
 * @param error - The error object to extract information from
 * @returns TRPCErrorInfo object or null if not a valid tRPC error
 */
export function extractTRPCErrorInfo(error: unknown): TRPCErrorInfo | null {
  if (!error || typeof error !== "object") {
    return null;
  }

  const errorObj = error as Record<string, unknown>;

  // Handle client-side tRPC errors
  if ("data" in errorObj && errorObj.data && typeof errorObj.data === "object") {
    const data = errorObj.data as Record<string, unknown>;
    if ("code" in data && typeof data.code === "string") {
      return {
        code: data.code,
        message: (errorObj.message as string) || (data.message as string) || "Unknown error",
        data: data,
        path: data.path as string,
      };
    }
  }

  // Handle server-side tRPC errors
  if ("code" in errorObj && typeof errorObj.code === "string") {
    return {
      code: errorObj.code,
      message: (errorObj.message as string) || "Unknown error",
      data: errorObj.data as Record<string, unknown>,
      path: errorObj.path as string,
    };
  }

  return null;
}

/**
 * Gets the appropriate log level for a tRPC error code
 *
 * @param code - The tRPC error code
 * @returns The log level to use for structured logging
 */
export function getErrorLogLevel(
  code: string,
): "trace" | "debug" | "info" | "warn" | "error" | "fatal" {
  return (
    (ERROR_SEVERITY_MAP as Record<string, "trace" | "debug" | "info" | "warn" | "error" | "fatal">)[
      code
    ] || "error"
  );
}

/**
 * Determines if an error should be reported to Sentry
 *
 * @param error - The error object to check
 * @returns true if the error should be reported to Sentry, false if it should be filtered
 */
export function shouldReportToSentry(error: unknown): boolean {
  return !isExpectedTRPCError(error);
}

/**
 * Creates a standardized error classification result
 */
export interface ErrorClassification {
  isExpected: boolean;
  shouldReport: boolean;
  errorInfo: TRPCErrorInfo | null;
  logLevel: "trace" | "debug" | "info" | "warn" | "error" | "fatal";
}

/**
 * Classifies an error and returns comprehensive classification information
 *
 * @param error - The error object to classify
 * @returns ErrorClassification object with all relevant information
 */
export function classifyError(error: unknown): ErrorClassification {
  const errorInfo = extractTRPCErrorInfo(error);
  const isExpected = isExpectedTRPCError(error);

  return {
    isExpected,
    shouldReport: !isExpected,
    errorInfo,
    logLevel: errorInfo ? getErrorLogLevel(errorInfo.code) : "error",
  };
}

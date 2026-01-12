/**
 * Structured Logger Wrapper for Sentry
 *
 * This module provides a structured logging wrapper around Sentry's logger
 * with consistent attribute naming, error severity mapping, and specialized
 * methods for tRPC error logging and user action logging.
 *
 */

import * as Sentry from "@sentry/nextjs";
import { type TRPCErrorInfo, getErrorLogLevel } from "../utils/error-classification";

/**
 * Base attributes included in all structured logs
 */
export interface BaseLogAttributes {
  service: "dashboard";
  version?: string;
  environment?: string;
  user_id?: string;
  workspace_id?: string;
  request_id?: string;
  [key: string]: string | number | boolean | undefined;
}

/**
 * tRPC-specific log attributes
 */
export interface TRPCLogAttributes extends BaseLogAttributes {
  trpc_procedure: string;
  trpc_error_code?: string;
  trpc_error_message?: string;
  response_time_ms?: number;
}

/**
 * User action log attributes
 */
export interface UserActionAttributes extends BaseLogAttributes {
  action_type: string;
  resource_type?: string;
  resource_id?: string;
}

/**
 * Generic log attributes for flexible logging
 */
export interface LogAttributes {
  [key: string]: string | number | boolean | undefined;
}

/**
 * Context information for logging operations
 */
export interface LogContext {
  userId?: string;
  workspaceId?: string;
  requestId?: string;
  timestamp: number; // Operation start time in milliseconds (Date.now()) - used for response time calculations
}

/**
 * Log levels supported by Sentry
 */
export type LogLevel = "trace" | "debug" | "info" | "warn" | "error" | "fatal";

/**
 * Creates base attributes for all log entries
 */
function createBaseAttributes(context?: Partial<LogContext>): BaseLogAttributes {
  return {
    service: "dashboard",
    version: process.env.NEXT_PUBLIC_APP_VERSION,
    environment: process.env.NODE_ENV,
    user_id: context?.userId,
    workspace_id: context?.workspaceId,
    request_id: context?.requestId,
  };
}
/**
 * Removes undefined values from an object to keep logs clean
 *
 * @param obj - The object to clean
 * @returns A new object with undefined values removed
 */
function removeUndefinedValues<T extends Record<string, unknown>>(obj: T): Partial<T> {
  const cleaned: Partial<T> = {};

  for (const [key, value] of Object.entries(obj)) {
    if (value !== undefined) {
      (cleaned as Record<string, unknown>)[key] = value;
    }
  }

  return cleaned;
}

/**
 * Logs a tRPC error with structured attributes
 *
 * Uses appropriate log level based on error severity mapping
 * and includes comprehensive context for debugging.
 *
 * @param errorInfo - Extracted tRPC error information
 * @param context - Additional context for the log entry
 */
export function logTRPCError(errorInfo: TRPCErrorInfo, context: LogContext): void {
  const level = getErrorLogLevel(errorInfo.code);
  const attributes: TRPCLogAttributes = {
    ...createBaseAttributes(context),
    trpc_procedure: errorInfo.path || "unknown",
    trpc_error_code: errorInfo.code,
    trpc_error_message: errorInfo.message,
    response_time_ms: context.timestamp ? Date.now() - context.timestamp : undefined,
  };

  // Remove undefined values to keep logs clean
  const cleanAttributes = removeUndefinedValues(attributes);

  try {
    // Guard against Sentry not being initialized or logger not available
    if (Sentry?.logger && typeof Sentry.logger[level] === "function") {
      Sentry.logger[level]("tRPC operation completed with expected error", cleanAttributes);
    } else {
      // Fallback to console if Sentry is disabled or not available
      console.debug(
        `[${level.toUpperCase()}] tRPC operation completed with expected error`,
        cleanAttributes,
      );
    }
  } catch (error) {
    // Fallback to console if Sentry fails
    console.error("Failed to log to Sentry:", error, {
      level,
      message: "tRPC operation completed with expected error",
      attributes: cleanAttributes,
    });
  }
}

/**
 * Logs a successful tRPC operation
 *
 * @param procedure - The tRPC procedure name
 * @param context - Additional context for the log entry
 * @param responseTimeMs - Optional response time in milliseconds
 */
export function logTRPCSuccess(
  procedure: string,
  context: LogContext,
  responseTimeMs?: number,
): void {
  const attributes: TRPCLogAttributes = {
    ...createBaseAttributes(context),
    trpc_procedure: procedure,
    response_time_ms: responseTimeMs,
  };

  const cleanAttributes = removeUndefinedValues(attributes);

  try {
    if (Sentry?.logger && typeof Sentry.logger.info === "function") {
      Sentry.logger.info("tRPC operation completed successfully", cleanAttributes);
    }
  } catch (error) {
    console.error("Failed to log to Sentry:", error, {
      message: "tRPC operation completed successfully",
      attributes: cleanAttributes,
    });
  }
}

/**
 * Logs a user action with structured attributes
 *
 * @param action - The action type being performed
 * @param userId - The ID of the user performing the action
 * @param attributes - Additional attributes for the action
 */
export function logUserAction(
  action: string,
  userId: string,
  attributes: Partial<UserActionAttributes> = {},
): void {
  const { user_id: _, ...safeAttributes } = attributes; // Exclude user_id from spread to prevent override
  const logAttributes: UserActionAttributes = {
    ...createBaseAttributes({ userId }),
    ...safeAttributes,
    action_type: action,
    user_id: userId, // Ensure this cannot be overridden
  };

  const cleanAttributes = removeUndefinedValues(logAttributes);

  try {
    if (Sentry?.logger && typeof Sentry.logger.info === "function") {
      Sentry.logger.info("User action performed", cleanAttributes);
    }
  } catch (error) {
    console.error("Failed to log to Sentry:", error, {
      message: "User action performed",
      attributes: cleanAttributes,
    });
  }
}

/**
 * Logs a generic operation with custom attributes
 *
 * @param level - The log level to use
 * @param message - The log message
 * @param attributes - Custom attributes for the log entry
 * @param context - Optional context information
 */
export function logOperation(
  level: LogLevel,
  message: string,
  attributes: LogAttributes = {},
  context?: Partial<LogContext>,
): void {
  const logAttributes = {
    ...createBaseAttributes(context),
    ...attributes,
  };

  const cleanAttributes = removeUndefinedValues(logAttributes);

  try {
    if (Sentry?.logger && typeof Sentry.logger[level] === "function") {
      Sentry.logger[level](message, cleanAttributes);
    }
  } catch (error) {
    console.error("Failed to log to Sentry:", error, {
      level,
      message,
      attributes: cleanAttributes,
    });
  }
}

/**
 * Logs request context information
 *
 * Useful for tracking request flow and debugging issues
 *
 * @param requestInfo - Information about the request
 * @param context - Additional context for the log entry
 */
export function logRequestContext(
  requestInfo: {
    method?: string;
    url?: string;
    userAgent?: string;
    ip?: string;
  },
  context: LogContext,
): void {
  const attributes = {
    ...createBaseAttributes(context),
    request_method: requestInfo.method,
    request_url: requestInfo.url,
    request_user_agent: requestInfo.userAgent,
    request_ip: requestInfo.ip,
  };

  const cleanAttributes = removeUndefinedValues(attributes);

  try {
    if (Sentry?.logger && typeof Sentry.logger.debug === "function") {
      Sentry.logger.debug("Request context", cleanAttributes);
    }
  } catch (error) {
    console.error("Failed to log to Sentry:", error, {
      message: "Request context",
      attributes: cleanAttributes,
    });
  }
}

/**
 * Logs performance metrics
 *
 * @param operation - The operation being measured
 * @param durationMs - Duration in milliseconds
 * @param context - Additional context for the log entry
 * @param additionalMetrics - Additional performance metrics
 */
export function logPerformance(
  operation: string,
  durationMs: number,
  context: LogContext,
  additionalMetrics: Record<string, number> = {},
): void {
  const attributes = {
    ...createBaseAttributes(context),
    operation_name: operation,
    duration_ms: durationMs,
    ...additionalMetrics,
  };

  const cleanAttributes = removeUndefinedValues(attributes);

  try {
    if (Sentry?.logger && typeof Sentry.logger.info === "function") {
      Sentry.logger.info("Performance metric", cleanAttributes);
    }
  } catch (error) {
    console.error("Failed to log to Sentry:", error, {
      message: "Performance metric",
      attributes: cleanAttributes,
    });
  }
}

/**
 * Creates a request ID for correlation
 *
 * @returns A unique request ID
 */
export function generateRequestId(): string {
  return `req_${Date.now()}_${Math.random().toString(36).substring(2, 11)}`;
}

/**
 * Gets current user context from various sources
 *
 * This is a helper method that can be extended to extract
 * user context from different sources (Clerk, session, etc.)
 *
 * @returns Current user context or undefined
 */
function getCurrentUserContext(): { userId?: string; workspaceId?: string } | undefined {
  // This would typically integrate with your auth system
  // For now, return undefined - can be extended based on auth implementation
  return undefined;
}

/**
 * Creates a log context with current timestamp and optional user context
 *
 * @param overrides - Optional context overrides
 * @returns A complete log context
 */
export function createLogContext(overrides: Partial<LogContext> = {}): LogContext {
  const userContext = getCurrentUserContext();

  return {
    timestamp: Date.now(),
    userId: userContext?.userId,
    workspaceId: userContext?.workspaceId,
    requestId: generateRequestId(),
    ...overrides,
  };
}

/**
 * Convenience functions for common logging patterns
 */

/**
 * Quick function to log a tRPC error with minimal setup
 */
export function logTRPCErrorQuick(errorInfo: TRPCErrorInfo, context?: Partial<LogContext>): void {
  const fullContext = createLogContext(context);
  logTRPCError(errorInfo, fullContext);
}

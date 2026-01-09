/**
 * Logging module exports
 *
 * Provides structured logging capabilities for the dashboard application
 * with consistent attribute naming and Sentry integration.
 */

export {
  logTRPCError,
  logUserAction,
  logOperation,
  type BaseLogAttributes,
  type TRPCLogAttributes,
  type UserActionAttributes,
  type LogAttributes,
  type LogContext,
  type LogLevel,
} from "./structured-logger";

// Re-export error classification utilities for convenience
export {
  isExpectedTRPCError,
  extractTRPCErrorInfo,
  getErrorLogLevel,
  classifyError,
  type TRPCErrorInfo,
  type ErrorClassification,
} from "../utils/error-classification";

/**
 * Logging module exports
 *
 * Provides structured logging capabilities for the dashboard application
 * with consistent attribute naming and Sentry integration.
 */

// Re-export error classification utilities for convenience
export {
  classifyError,
  type ErrorClassification,
  extractTRPCErrorInfo,
  getErrorLogLevel,
  isExpectedTRPCError,
  type TRPCErrorInfo,
} from "../utils/error-classification";
export {
  type BaseLogAttributes,
  type LogAttributes,
  type LogContext,
  type LogLevel,
  logOperation,
  logTRPCError,
  logUserAction,
  type TRPCLogAttributes,
  type UserActionAttributes,
} from "./structured-logger";

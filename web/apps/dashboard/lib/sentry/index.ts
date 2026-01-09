/**
 * Sentry Integration Module
 *
 * This module provides a centralized API for Sentry-related functionality
 * including error filtering, structured logging, and configuration utilities.
 */

// Export error filtering functionality
export {
  createErrorFilter,
  createClientErrorFilter,
  createServerErrorFilter,
  createEdgeErrorFilter,
  shouldReportError,
  logFilteredError,
  preserveErrorContext,
  hasPreservedContext,
  getPreservedContext,
  type ErrorFilterOptions,
  type BeforeSendHook,
} from "./error-filter";

// Re-export error classification utilities for convenience
export {
  isExpectedTRPCError,
  extractTRPCErrorInfo,
  classifyError,
  isExpectedErrorCode,
  getErrorLogLevel,
  shouldReportToSentry,
  EXPECTED_TRPC_CODES,
  ERROR_SEVERITY_MAP,
  type TRPCErrorInfo,
  type ErrorClassification,
} from "../utils/error-classification";

// Re-export structured logging for convenience
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
} from "../logging/structured-logger";

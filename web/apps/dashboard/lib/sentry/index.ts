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

// Export trace sampling functionality
export { createTracesSampler } from "./trace-sampler";

// Export PII/URL scrubbing utilities. `scrubLog` and `scrubReplayFrame` cover
// the two egress paths that bypass `beforeSend` entirely — structured logs and
// Session Replay recording frames.
export {
  scrubUrl,
  scrubEventPii,
  scrubTransactionPii,
  scrubSpanPii,
  scrubLog,
  scrubReplayFrame,
} from "./pii-scrubber";

// Export Replay privacy config and noise filters
export { replayPrivacyOptions } from "./replay-privacy";
export { IGNORE_ERRORS, DENY_URLS } from "./noise-filters";

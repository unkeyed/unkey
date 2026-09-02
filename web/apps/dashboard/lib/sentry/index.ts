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
} from "../logging/structured-logger";

export {
  classifyError,
  ERROR_SEVERITY_MAP,
  type ErrorClassification,
  EXPECTED_TRPC_CODES,
  extractTRPCErrorInfo,
  getErrorLogLevel,
  isExpectedErrorCode,
  isExpectedTRPCError,
  shouldReportToSentry,
  type TRPCErrorInfo,
} from "../utils/error-classification";
export {
  type BeforeSendHook,
  createClientErrorFilter,
  createEdgeErrorFilter,
  createErrorFilter,
  createServerErrorFilter,
  type ErrorFilterOptions,
  getPreservedContext,
  hasPreservedContext,
  logFilteredError,
  preserveErrorContext,
  shouldReportError,
} from "./error-filter";
export { DENY_URLS, IGNORE_ERRORS } from "./noise-filters";

export {
  scrubEventPii,
  scrubLog,
  scrubReplayFrame,
  scrubSpanPii,
  scrubTransactionPii,
  scrubUrl,
} from "./pii-scrubber";

export { replayPrivacyOptions } from "./replay-privacy";
export { createTracesSampler } from "./trace-sampler";

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

export { createTracesSampler } from "./trace-sampler";

export {
  scrubUrl,
  scrubEventPii,
  scrubTransactionPii,
  scrubSpanPii,
  scrubLog,
  scrubReplayFrame,
} from "./pii-scrubber";

export { replayPrivacyOptions } from "./replay-privacy";
export { IGNORE_ERRORS, DENY_URLS } from "./noise-filters";

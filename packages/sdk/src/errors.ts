export type ErrorCode =
  | "NOT_FOUND"
  | "BAD_REQUEST"
  | "UNAUTHORIZED"
  | "INTERNAL_SERVER_ERROR"
  | "RATELIMITED"
  | "FORBIDDEN"
  | "KEY_USAGE_EXCEEDED"
  | "INVALID_KEY_TYPE"
  | "NOT_UNIQUE";

export type UnkeyError = {
  // A machine readable error code
  code: ErrorCode;

  // A link to our documentation explaining this error in more detail
  docs: string;

  // A human readable short explanation
  message: string;

  // The request id for easy support lookup
  requestId: string;
};

// this is what a json body response looks like
export type ErrorResponse = {
  error: UnkeyError;
};

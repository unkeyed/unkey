import { z } from "@hono/zod-openapi";
import type { paths } from "./openapi";

export const ErrorCode = z.enum([
  "BAD_REQUEST",
  "FORBIDDEN",
  "INTERNAL_SERVER_ERROR",
  "USAGE_EXCEEDED",
  "DISABLED",
  "NOT_FOUND",
  "NOT_UNIQUE",
  "RATE_LIMITED",
  "UNAUTHORIZED",
  "PRECONDITION_FAILED",
  "INSUFFICIENT_PERMISSIONS",
  "METHOD_NOT_ALLOWED",
]);

export type ErrorCodeType = z.infer<typeof ErrorCode>;

// this is what a json body response looks like
export type ErrorResponse =
  paths["/v1/liveness"]["get"]["responses"]["500"]["content"]["application/json"];

import type { paths } from "./openapi";
export type { ErrorCodeType } from "../../../apps/api/src/pkg/errors/http";

// this is what a json body response looks like
export type ErrorResponse =
  paths["/v1/liveness"]["get"]["responses"]["500"]["content"]["application/json"];

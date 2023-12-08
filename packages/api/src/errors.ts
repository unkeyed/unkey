import { paths } from "./openapi";

// this is what a json body response looks like
export type ErrorResponse =
  paths["/v1/liveness"]["get"]["responses"]["500"]["content"]["application/json"];

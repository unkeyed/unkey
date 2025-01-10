import type { paths } from "./openapi";

// this is what a json body response looks like
export type ErrorResponse =
  | paths["/v1/liveness"]["get"]["responses"]["400"]["content"]["application/json"]
  | paths["/v1/liveness"]["get"]["responses"]["401"]["content"]["application/json"]
  | paths["/v1/liveness"]["get"]["responses"]["403"]["content"]["application/json"]
  | paths["/v1/liveness"]["get"]["responses"]["404"]["content"]["application/json"]
  | paths["/v1/liveness"]["get"]["responses"]["409"]["content"]["application/json"]
  | paths["/v1/liveness"]["get"]["responses"]["429"]["content"]["application/json"]
  | paths["/v1/liveness"]["get"]["responses"]["500"]["content"]["application/json"];

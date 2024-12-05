export type Log = {
  request_id: string;
  time: number;
  workspace_id: string;
  host: string;
  method: string;
  path: string;
  request_headers: string[];
  request_body: string;
  response_status: number;
  response_headers: string[];
  response_body: string;
  error: string;
  service_latency: number;
};

export type ResponseBody = {
  keyId: string;
  valid: boolean;
  meta: Record<string, unknown>;
  enabled: boolean;
  permissions: string[];
  code:
    | "VALID"
    | "RATE_LIMITED"
    | "EXPIRED"
    | "USAGE_EXCEEDED"
    | "DISABLED"
    | "FORBIDDEN"
    | "INSUFFICIENT_PERMISSIONS";
};

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

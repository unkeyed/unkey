import type { RateLimitIdentifierSource } from "./model";

type IdentifierSource = {
  label: string;
  short: string;
  phrase: string;
  value: { placeholder: string; required: string } | null;
  needsIdentity: boolean;
};

export const IDENTIFIER_SOURCES: Record<RateLimitIdentifierSource, IdentifierSource> = {
  remoteIp: {
    label: "Client IP",
    short: "client IP",
    phrase: "client IP",
    value: null,
    needsIdentity: false,
  },
  header: {
    label: "Header",
    short: "header",
    phrase: "header value",
    value: { placeholder: "X-Tenant-Id", required: "Header name is required" },
    needsIdentity: false,
  },
  authenticatedSubject: {
    label: "Authenticated Subject",
    short: "subject",
    phrase: "authenticated subject",
    value: null,
    needsIdentity: true,
  },
  path: {
    label: "Request Path",
    short: "path",
    phrase: "request path",
    value: null,
    needsIdentity: false,
  },
  principalField: {
    label: "Principal Field",
    short: "principal field",
    phrase: "principal field value",
    value: { placeholder: "subject", required: "Field path is required" },
    needsIdentity: true,
  },
};

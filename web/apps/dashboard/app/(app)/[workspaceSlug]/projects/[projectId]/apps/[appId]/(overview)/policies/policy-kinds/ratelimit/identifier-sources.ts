import type { RateLimitIdentifierSource } from "./model";

type IdentifierSource = {
  label: string;
  short: string;
  phrase: string;
  valuePlaceholder: string | null;
  needsIdentity: boolean;
};

export const IDENTIFIER_SOURCES: Record<RateLimitIdentifierSource, IdentifierSource> = {
  remoteIp: {
    label: "Client IP",
    short: "client IP",
    phrase: "client IP",
    valuePlaceholder: null,
    needsIdentity: false,
  },
  header: {
    label: "Header",
    short: "header",
    phrase: "header value",
    valuePlaceholder: "X-Tenant-Id",
    needsIdentity: false,
  },
  authenticatedSubject: {
    label: "Authenticated Subject",
    short: "subject",
    phrase: "authenticated subject",
    valuePlaceholder: null,
    needsIdentity: true,
  },
  path: {
    label: "Request Path",
    short: "path",
    phrase: "request path",
    valuePlaceholder: null,
    needsIdentity: false,
  },
  principalField: {
    label: "Principal Field",
    short: "principal field",
    phrase: "principal field value",
    valuePlaceholder: "subject",
    needsIdentity: true,
  },
};

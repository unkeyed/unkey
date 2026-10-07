import type { StringMatchMode } from "@/lib/collections/deploy/policies.schema";
import type { MatchConditionFormValues } from "../schema";

export const HTTP_METHODS = ["GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"] as const;
export type HttpMethod = (typeof HTTP_METHODS)[number];

export const STRING_MATCH_MODES: { value: StringMatchMode; label: string }[] = [
  { value: "exact", label: "Exact" },
  { value: "prefix", label: "Prefix" },
  { value: "regex", label: "Regex" },
];

export const MATCH_TYPE_OPTIONS: { value: MatchConditionFormValues["type"]; label: string }[] = [
  { value: "path", label: "Path" },
  { value: "method", label: "Method" },
  { value: "header", label: "Header" },
  { value: "queryParam", label: "Query Param" },
  { value: "remoteIp", label: "Remote IP" },
];

export const REMOTE_IP_OPERATORS: { value: "in" | "notIn"; label: string }[] = [
  { value: "in", label: "In range" },
  { value: "notIn", label: "Not in range" },
];

export function validateRegexSyntax(pattern: string): string | undefined {
  if (!pattern) {
    return undefined;
  }
  try {
    new RegExp(pattern);
    return undefined;
  } catch (e) {
    return e instanceof SyntaxError ? e.message : "Invalid regex pattern";
  }
}

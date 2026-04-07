import type { StringMatchMode } from "@/lib/trpc/routers/deploy/environment-settings/sentinel/update-middleware";
import type { MatchConditionFormValues } from "../schema";

export const HTTP_METHODS = ["GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"];

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

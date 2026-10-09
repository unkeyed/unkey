import type { ValidationResult } from "./use-repo-tree";

export type PathHint =
  | { type: "none" }
  | { type: "case-match"; path: string }
  | { type: "not-found"; branch: string | null };

export type PathInputVariant = "error" | "warning" | "default";

export function pathHint(
  validation: ValidationResult,
  findCaseMatch: () => string | null,
  branch: string | null,
): PathHint {
  if (validation !== "invalid") {
    return { type: "none" };
  }
  const caseMatch = findCaseMatch();
  if (caseMatch) {
    return { type: "case-match", path: caseMatch };
  }
  return { type: "not-found", branch };
}

export function pathInputVariant(hasError: boolean, hint: PathHint): PathInputVariant {
  if (hasError) {
    return "error";
  }
  return hint.type === "none" ? "default" : "warning";
}

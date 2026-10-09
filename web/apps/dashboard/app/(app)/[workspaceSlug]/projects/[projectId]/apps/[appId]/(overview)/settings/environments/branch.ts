import type { Environment } from "@/lib/collections/deploy/environments";

export type Branch = { kind: "default"; name: string } | { kind: "unassigned" } | { kind: "none" };

export function branchFor(
  environment: Pick<Environment, "kind">,
  hasRepository: boolean,
  defaultBranch: string | null,
): Branch {
  if (!hasRepository) {
    return { kind: "none" };
  }
  if (environment.kind === "preview") {
    return { kind: "unassigned" };
  }
  return defaultBranch ? { kind: "default", name: defaultBranch } : { kind: "none" };
}

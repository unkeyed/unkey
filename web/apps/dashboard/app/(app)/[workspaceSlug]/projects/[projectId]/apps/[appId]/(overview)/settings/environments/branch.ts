import type { Environment } from "@/lib/collections/deploy/environments";

export type Branch = { kind: "default"; name: string } | { kind: "unassigned" } | { kind: "none" };

function trackedBranch(
  environment: Pick<Environment, "kind">,
  defaultBranch: string | null,
): Branch {
  if (environment.kind === "preview") {
    return { kind: "unassigned" };
  }
  return defaultBranch ? { kind: "default", name: defaultBranch } : { kind: "none" };
}

export function branchFor(
  environment: Pick<Environment, "kind">,
  hasRepository: boolean,
  defaultBranch: string | null,
): Branch {
  return hasRepository ? trackedBranch(environment, defaultBranch) : { kind: "none" };
}

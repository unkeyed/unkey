import { policyMatchKey } from "@/lib/collections/deploy/policies.schema";
import { POLICY_KINDS, type PolicyType } from "../policy-kinds";
import type { MergedPolicy } from "./merge";

type Named = { type: PolicyType; name: string };

export function otherMatchKeys(
  merged: readonly MergedPolicy[],
  exceptKey: string | null,
): string[] {
  return merged.filter((m) => m.key !== exceptKey).map((m) => policyMatchKey(m.type, m.name));
}

export function duplicateNameError(next: Named, otherKeys: readonly string[]): string | null {
  if (!otherKeys.includes(policyMatchKey(next.type, next.name))) {
    return null;
  }
  return `A ${POLICY_KINDS[next.type].label} policy named "${next.name}" already exists. Open it to add it to another environment.`;
}

import { policyMatchKey } from "@/lib/collections/deploy/policies.schema";
import { POLICY_KINDS, type PolicyType } from "../../policy-kinds";

type Named = { type: PolicyType; name: string };

export function duplicateNameError(
  next: Named,
  existingMatchKeys: readonly string[],
  current: Named | null,
): string | null {
  const nextKey = policyMatchKey(next.type, next.name);
  const currentKey = current ? policyMatchKey(current.type, current.name) : null;
  if (nextKey === currentKey || !existingMatchKeys.includes(nextKey)) {
    return null;
  }
  return `A ${POLICY_KINDS[next.type].label} policy named "${next.name}" already exists. Open it to add it to another environment.`;
}

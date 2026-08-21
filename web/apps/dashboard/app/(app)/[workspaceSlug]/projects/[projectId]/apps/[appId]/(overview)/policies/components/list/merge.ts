import type { PolicyRow } from "@/lib/collections/deploy/policies";
import {
  type Policy,
  normalizePolicyName,
  policyIdentity,
} from "@/lib/collections/deploy/policies.schema";

export type Env = "production" | "preview";

/**
 * One row of the merged list. It holds up to two environment copies of one
 * policy.
 *
 * `key` identifies the row. It is the policy identity for most rows, because
 * each copy has its own server id and the identity is the only value they
 * share. If an identity occurs more than one time in one environment, the key
 * uses an id. `key` is opaque: read a policy from it with `policyInEnv`, and
 * render `name` instead.
 */
export type MergedPolicy = {
  key: string;
  name: string;
  type: Policy["type"];
  production: PolicyRow | null;
  preview: PolicyRow | null;
};

export function mergePolicies(production: PolicyRow[], preview: PolicyRow[]): MergedPolicy[] {
  const inProduction = countByIdentity(production);
  const inPreview = countByIdentity(preview);

  const pairable = (p: PolicyRow, counts: Map<string, number>) =>
    normalizePolicyName(p.name).length > 0 && counts.get(policyIdentity(p.type, p.name)) === 1;

  const pairablePreview = new Map(
    preview.filter((p) => pairable(p, inPreview)).map((p) => [policyIdentity(p.type, p.name), p]),
  );
  const pairedIdentities = new Set<string>();

  const result: MergedPolicy[] = production.map((p) => {
    const identity = policyIdentity(p.type, p.name);
    const unique = pairable(p, inProduction);
    const partner = unique ? pairablePreview.get(identity) : undefined;
    if (partner) {
      pairedIdentities.add(identity);
    }
    return {
      key: unique ? identity : `production:${p.id}`,
      name: p.name,
      type: p.type,
      production: p,
      preview: partner ?? null,
    };
  });

  for (const p of preview) {
    const identity = policyIdentity(p.type, p.name);
    if (pairedIdentities.has(identity)) {
      continue;
    }
    result.push({
      key: pairable(p, inPreview) ? identity : `preview:${p.id}`,
      name: p.name,
      type: p.type,
      production: null,
      preview: p,
    });
  }

  return result;
}

function countByIdentity(policies: PolicyRow[]): Map<string, number> {
  const counts = new Map<string, number>();
  for (const p of policies) {
    const identity = policyIdentity(p.type, p.name);
    counts.set(identity, (counts.get(identity) ?? 0) + 1);
  }
  return counts;
}

export function policyInEnv(merged: MergedPolicy[], key: string, env: Env): PolicyRow | null {
  return merged.find((m) => m.key === key)?.[env] ?? null;
}

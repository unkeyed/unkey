import type { EnvironmentKind } from "@/lib/collections/deploy/environments";
import type { PolicyRow } from "@/lib/collections/deploy/policies";
import {
  normalizePolicyName,
  type Policy,
  policyMatchKey,
} from "@/lib/collections/deploy/policies.schema";

export type Env = EnvironmentKind;

/**
 * One row of the merged list. It holds up to two environment copies of one
 * policy.
 *
 * `key` identifies the row. It is the policy match key for most rows, because
 * each copy has its own server id and the match key is the only value they
 * share. If a match key occurs more than one time in one environment, the key
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
  const inProduction = countByMatchKey(production);
  const inPreview = countByMatchKey(preview);

  const pairable = (p: PolicyRow, counts: Map<string, number>) =>
    normalizePolicyName(p.name).length > 0 && counts.get(policyMatchKey(p.type, p.name)) === 1;

  const pairablePreview = new Map(
    preview.filter((p) => pairable(p, inPreview)).map((p) => [policyMatchKey(p.type, p.name), p]),
  );
  const pairedMatchKeys = new Set<string>();

  const result: MergedPolicy[] = production.map((p) => {
    const matchKey = policyMatchKey(p.type, p.name);
    const unique = pairable(p, inProduction);
    const partner = unique ? pairablePreview.get(matchKey) : undefined;
    if (partner) {
      pairedMatchKeys.add(matchKey);
    }
    return {
      key: unique ? matchKey : `production:${p.id}`,
      name: p.name,
      type: p.type,
      production: p,
      preview: partner ?? null,
    };
  });

  for (const p of preview) {
    const matchKey = policyMatchKey(p.type, p.name);
    if (pairedMatchKeys.has(matchKey)) {
      continue;
    }
    result.push({
      key: pairable(p, inPreview) ? matchKey : `preview:${p.id}`,
      name: p.name,
      type: p.type,
      production: null,
      preview: p,
    });
  }

  return result;
}

function countByMatchKey(policies: PolicyRow[]): Map<string, number> {
  const counts = new Map<string, number>();
  for (const p of policies) {
    const matchKey = policyMatchKey(p.type, p.name);
    counts.set(matchKey, (counts.get(matchKey) ?? 0) + 1);
  }
  return counts;
}

export function policyInEnv(merged: MergedPolicy[], key: string, env: Env): PolicyRow | null {
  return merged.find((m) => m.key === key)?.[env] ?? null;
}

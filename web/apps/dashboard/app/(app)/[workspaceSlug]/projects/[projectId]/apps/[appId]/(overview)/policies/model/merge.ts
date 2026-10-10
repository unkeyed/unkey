import type { EnvironmentKind } from "@/lib/collections/deploy/environments";
import { type PolicyRow, rowKey } from "@/lib/collections/deploy/policies";
import {
  type Policy,
  normalizePolicyName,
  policyMatchKey,
} from "@/lib/collections/deploy/policies.schema";

export type Env = EnvironmentKind;

export type PolicyEnvs = Record<Env, { id: string | null; slug: string }>;

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
      key: unique ? matchKey : rowKey(p.environmentId, p.id),
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
      key: pairable(p, inPreview) ? matchKey : rowKey(p.environmentId, p.id),
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

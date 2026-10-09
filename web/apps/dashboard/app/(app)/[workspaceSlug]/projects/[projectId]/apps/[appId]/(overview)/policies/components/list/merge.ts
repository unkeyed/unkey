import type { EnvironmentKind } from "@/lib/collections/deploy/environments";
import type { PolicyRow } from "@/lib/collections/deploy/policies";
import {
  type Policy,
  normalizePolicyName,
  policyMatchKey,
} from "@/lib/collections/deploy/policies.schema";

export type Env = EnvironmentKind;

export type PolicyEnvs = Record<Env, { id: string | null; slug: string }>;

declare const policyRowKeyBrand: unique symbol;

export type PolicyRowKey = string & { readonly [policyRowKeyBrand]: true };

export type MergedPolicy = {
  key: PolicyRowKey;
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

  const result: MergedPolicy[] = production.map((p, index) => {
    const matchKey = policyMatchKey(p.type, p.name);
    const unique = pairable(p, inProduction);
    const partner = unique ? pairablePreview.get(matchKey) : undefined;
    if (partner) {
      pairedMatchKeys.add(matchKey);
    }
    return {
      key: rowKey(unique ? matchKey : positionKey("production", index, matchKey)),
      name: p.name,
      type: p.type,
      production: p,
      preview: partner ?? null,
    };
  });

  for (const [index, p] of preview.entries()) {
    const matchKey = policyMatchKey(p.type, p.name);
    if (pairedMatchKeys.has(matchKey)) {
      continue;
    }
    result.push({
      key: rowKey(pairable(p, inPreview) ? matchKey : positionKey("preview", index, matchKey)),
      name: p.name,
      type: p.type,
      production: null,
      preview: p,
    });
  }

  return result;
}

const rowKey = (key: string) => key as PolicyRowKey;

const positionKey = (env: Env, index: number, matchKey: string) => `${env}#${index}#${matchKey}`;

function countByMatchKey(policies: PolicyRow[]): Map<string, number> {
  const counts = new Map<string, number>();
  for (const p of policies) {
    const matchKey = policyMatchKey(p.type, p.name);
    counts.set(matchKey, (counts.get(matchKey) ?? 0) + 1);
  }
  return counts;
}

export function policyInEnv(merged: MergedPolicy[], key: PolicyRowKey, env: Env): PolicyRow | null {
  return merged.find((m) => m.key === key)?.[env] ?? null;
}

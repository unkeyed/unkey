import type { EnvironmentKind } from "@/lib/collections/deploy/environments";
import type { PolicyRow } from "@/lib/collections/deploy/policies";
import {
  type Policy,
  normalizePolicyName,
  policyMatchKey,
} from "@/lib/collections/deploy/policies.schema";

export type Env = EnvironmentKind;

/** `id` is empty while the environment is unknown. */
export type PolicyEnvs = Record<Env, { id: string; slug: string }>;

/**
 * One row of the merged list. It holds up to two environment copies of one
 * policy.
 *
 * `key` identifies the row. It is the policy match key for most rows, because
 * each copy has its own server id and the match key is the only value they
 * share. A row whose name is blank or repeats in its environment is keyed by
 * its environment, position and match key instead. Ids change on every write,
 * so an id key would not find the row for an edit queued behind another one.
 * `key` is opaque: read a policy from it with `policyInEnv`, and render `name`
 * instead.
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

  const result: MergedPolicy[] = production.map((p, index) => {
    const matchKey = policyMatchKey(p.type, p.name);
    const unique = pairable(p, inProduction);
    const partner = unique ? pairablePreview.get(matchKey) : undefined;
    if (partner) {
      pairedMatchKeys.add(matchKey);
    }
    return {
      key: unique ? matchKey : positionKey("production", index, matchKey),
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
      key: pairable(p, inPreview) ? matchKey : positionKey("preview", index, matchKey),
      name: p.name,
      type: p.type,
      production: null,
      preview: p,
    });
  }

  return result;
}

const positionKey = (env: Env, index: number, matchKey: string) => `${env}#${index}#${matchKey}`;

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

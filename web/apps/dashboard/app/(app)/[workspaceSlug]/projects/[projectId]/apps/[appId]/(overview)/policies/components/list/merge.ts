import type { PolicyRow } from "@/lib/collections/deploy/policies";
import {
  type Policy,
  normalizePolicyName,
  policyIdentity,
} from "@/lib/collections/deploy/policies.schema";

/**
 * One row of the merged list. It holds up to two environment copies of one
 * policy. envA is production and envB is preview.
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
  envA: PolicyRow | null;
  envB: PolicyRow | null;
};

/**
 * Merges the two environment lists into one row for each policy identity. The
 * order follows envA. Rows that exist only in envB come last.
 *
 * An identity pairs the two environments only if it occurs one time in each
 * list. An identity that occurs more than one time in an environment gets an
 * id key. The add panel refuses such a name, but the API accepts it. The id
 * key keeps one edit from changing two different policies.
 */
export function mergePolicies(a: PolicyRow[], b: PolicyRow[]): MergedPolicy[] {
  const countA = countByIdentity(a);
  const countB = countByIdentity(b);
  // A blank name is not an identity. It must not pair two different policies.
  // It also cannot be a React key.
  const pairable = (p: PolicyRow, counts: Map<string, number>) =>
    normalizePolicyName(p.name).length > 0 && counts.get(policyIdentity(p.type, p.name)) === 1;

  const pairableB = new Map(
    b.filter((p) => pairable(p, countB)).map((p) => [policyIdentity(p.type, p.name), p]),
  );
  const pairedIdentities = new Set<string>();

  const result: MergedPolicy[] = a.map((p) => {
    const identity = policyIdentity(p.type, p.name);
    const uniqueInA = pairable(p, countA);
    const partner = uniqueInA ? pairableB.get(identity) : undefined;
    if (partner) {
      pairedIdentities.add(identity);
    }
    return {
      key: uniqueInA ? identity : `envA:${p.id}`,
      name: p.name,
      type: p.type,
      envA: p,
      envB: partner ?? null,
    };
  });

  for (const p of b) {
    const identity = policyIdentity(p.type, p.name);
    if (pairedIdentities.has(identity)) {
      continue;
    }
    result.push({
      key: pairable(p, countB) ? identity : `envB:${p.id}`,
      name: p.name,
      type: p.type,
      envA: null,
      envB: p,
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

/**
 * Returns the copy of a merged row in one environment, or null.
 *
 * Use this function to read a policy from a `MergedPolicy.key`. A key is an
 * identity for most rows and an id for a duplicate one. A lookup by name alone
 * finds no duplicate row.
 */
export function policyInEnv(
  merged: MergedPolicy[],
  key: string,
  env: "envA" | "envB",
): PolicyRow | null {
  return merged.find((m) => m.key === key)?.[env] ?? null;
}

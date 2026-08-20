import type { PolicyRow } from "@/lib/collections/deploy/policies";
import type { Policy } from "@/lib/collections/deploy/policies.schema";

/**
 * One row of the merged list. It holds up to two environment copies of one
 * policy. envA is production and envB is preview.
 *
 * `key` identifies the row. It is the policy name for most rows, because each
 * copy has its own server id and the name is the only value they share. If a
 * name occurs more than one time in one environment, the key uses an id.
 * Use `policyInEnv` to read a policy from a key. Do not match a key on a name.
 */
export type MergedPolicy = {
  key: string;
  name: string;
  type: Policy["type"];
  envA: PolicyRow | null;
  envB: PolicyRow | null;
};

/**
 * Merges the two environment lists into one row for each policy name. The
 * order follows envA. Rows that exist only in envB come last.
 *
 * A name pairs the two environments only if it occurs one time in each list.
 * A name that occurs more than one time in an environment gets an id key.
 * The add panel refuses such a name, but the API accepts it. The id key keeps
 * one edit from changing two different policies.
 */
export function mergePolicies(a: PolicyRow[], b: PolicyRow[]): MergedPolicy[] {
  const countA = countByName(a);
  const countB = countByName(b);
  // A blank name is not an identity. It must not pair two different policies.
  // It also cannot be a React key.
  const pairable = (p: PolicyRow, counts: Map<string, number>) =>
    p.name.trim().length > 0 && counts.get(p.name) === 1;

  const pairableB = new Map(b.filter((p) => pairable(p, countB)).map((p) => [p.name, p]));
  const pairedNames = new Set<string>();

  const result: MergedPolicy[] = a.map((p) => {
    const uniqueInA = pairable(p, countA);
    const partner = uniqueInA ? pairableB.get(p.name) : undefined;
    if (partner) {
      pairedNames.add(p.name);
    }
    return {
      key: uniqueInA ? p.name : `envA:${p.id}`,
      name: p.name,
      type: p.type,
      envA: p,
      envB: partner ?? null,
    };
  });

  for (const p of b) {
    if (pairedNames.has(p.name)) {
      continue;
    }
    const uniqueInB = pairable(p, countB);
    result.push({
      key: uniqueInB ? p.name : `envB:${p.id}`,
      name: p.name,
      type: p.type,
      envA: null,
      envB: p,
    });
  }

  return result;
}

function countByName(policies: PolicyRow[]): Map<string, number> {
  const counts = new Map<string, number>();
  for (const p of policies) {
    counts.set(p.name, (counts.get(p.name) ?? 0) + 1);
  }
  return counts;
}

/**
 * Returns the copy of a merged row in one environment, or null.
 *
 * Use this function to read a policy from a `MergedPolicy.key`. A key is a
 * name for most rows and an id for a duplicate name. A lookup by name alone
 * finds no duplicate row.
 */
export function policyInEnv(
  merged: MergedPolicy[],
  key: string,
  env: "envA" | "envB",
): PolicyRow | null {
  return merged.find((m) => m.key === key)?.[env] ?? null;
}

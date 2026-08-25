import type { Policy } from "@/lib/collections/deploy/policies.schema";

/**
 * One row in the merged gateway policies list. Each row represents a single
 * logical policy id, paired with up to two env-specific instances (envA =
 * production, envB = preview by convention).
 */
export type MergedPolicy = {
  id: string;
  name: string;
  type: Policy["type"];
  envA: Policy | null;
  envB: Policy | null;
};

/**
 * Merge per-env policy lists into one row per logical policy id. Order
 * follows envA's order; envB-only policies are appended at the end.
 */
export function mergePolicies(a: Policy[], b: Policy[]): MergedPolicy[] {
  const mapB = new Map(b.map((p) => [p.id, p]));
  const seen = new Set<string>();
  const result: MergedPolicy[] = a.map((p) => {
    seen.add(p.id);
    return { id: p.id, name: p.name, type: p.type, envA: p, envB: mapB.get(p.id) ?? null };
  });
  for (const p of b) {
    if (!seen.has(p.id)) {
      result.push({ id: p.id, name: p.name, type: p.type, envA: null, envB: p });
    }
  }
  return result;
}

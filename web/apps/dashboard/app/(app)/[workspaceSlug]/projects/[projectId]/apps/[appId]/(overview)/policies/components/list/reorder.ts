import { ENVIRONMENT_KINDS } from "@/lib/collections/deploy/environments";
import type { PolicyRow } from "@/lib/collections/deploy/policies";
import type { Env, MergedPolicy } from "./merge";

type PolicyListOrder = { env: Env; rows: PolicyRow[] };

export function movePolicy(
  merged: MergedPolicy[],
  rowsByEnv: Record<Env, PolicyRow[]>,
  from: number,
  to: number,
): PolicyListOrder[] {
  const moved = merged[from];
  if (!moved || from === to) {
    return [];
  }
  const down = from < to;
  const passed = down ? merged.slice(from + 1, to + 1) : merged.slice(to, from);

  return ENVIRONMENT_KINDS.flatMap((env) => {
    const row = moved[env];
    if (!row) {
      return [];
    }
    const passedIds = new Set(passed.flatMap((m) => m[env]?.id ?? []));
    const list = rowsByEnv[env];
    const passedAt = list.flatMap((r, i) => (passedIds.has(r.id) ? [i] : []));
    const rowAt = list.findIndex((r) => r.id === row.id);
    const first = passedAt.at(0);
    const last = passedAt.at(-1);
    if (first === undefined || last === undefined) {
      return [];
    }
    if (down ? rowAt > last : rowAt < first) {
      return [];
    }
    const rest = list.filter((r) => r.id !== row.id);
    const at = down ? last : first;
    return [{ env, rows: [...rest.slice(0, at), row, ...rest.slice(at)] }];
  });
}

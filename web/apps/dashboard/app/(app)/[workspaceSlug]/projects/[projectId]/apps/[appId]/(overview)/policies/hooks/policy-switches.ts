import { match } from "@unkey/match";
import type { Env, MergedPolicy, PolicyRowKey } from "../components/list/merge";

export type PolicySwitches = Record<Env, boolean>;

type PendingSwitch = {
  on: boolean;
  click: number;
  /** The collection syncs a tick after the write resolves, so a settled entry stays until then. */
  settled: boolean;
};

export type PendingSwitches = ReadonlyMap<string, PendingSwitch>;

export const NO_PENDING_SWITCHES: PendingSwitches = new Map();

export type PendingSwitchEvent =
  | { type: "click"; key: PolicyRowKey; env: Env; on: boolean; click: number }
  | { type: "settle"; key: PolicyRowKey; env: Env; click: number; ok: boolean }
  | { type: "sync" };

const slot = (key: PolicyRowKey, env: Env) => `${env}:${key}`;

export function reducePendingSwitches(
  state: PendingSwitches,
  event: PendingSwitchEvent,
): PendingSwitches {
  return match(event)
    .with({ type: "click" }, (e) =>
      new Map(state).set(slot(e.key, e.env), { on: e.on, click: e.click, settled: false }),
    )
    .with({ type: "settle" }, (e) => {
      const id = slot(e.key, e.env);
      const entry = state.get(id);
      if (entry?.click !== e.click) {
        return state;
      }
      const next = new Map(state);
      if (e.ok) {
        next.set(id, { ...entry, settled: true });
      } else {
        next.delete(id);
      }
      return next;
    })
    .with({ type: "sync" }, () =>
      [...state.values()].some((entry) => entry.settled)
        ? new Map([...state].filter(([, entry]) => !entry.settled))
        : state,
    )
    .exhaustive();
}

export function policySwitches(policy: MergedPolicy, pending: PendingSwitches): PolicySwitches {
  const on = (env: Env) => pending.get(slot(policy.key, env))?.on ?? policy[env]?.enabled ?? false;
  return { production: on("production"), preview: on("preview") };
}

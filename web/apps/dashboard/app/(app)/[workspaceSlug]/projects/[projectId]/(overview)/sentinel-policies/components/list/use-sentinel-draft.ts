"use client";

import { collection } from "@/lib/collections";
import type { SentinelPolicy } from "@/lib/collections/deploy/sentinel-policies.schema";
import { useCallback, useMemo, useState } from "react";

export type SentinelDraftActions = {
  reorder: (next: MergedPolicy[]) => void;
  toggleEnv: (id: string, env: "envA" | "envB") => void;
  addToEnv: (id: string, env: "envA" | "envB") => void;
  saveConfig: (
    id: string,
    prodPolicy: SentinelPolicy,
    previewPolicy: SentinelPolicy | null,
  ) => void;
};

type Args = {
  envAId: string;
  envBId: string;
  policiesA: SentinelPolicy[];
  policiesB: SentinelPolicy[];
};

/**
 * Stages cheap, reversible sentinel-policy edits (toggle, reorder, config
 * tweaks) locally. While `draft` is null the UI mirrors the server; the first
 * edit forks a draft and subsequent actions mutate it. Save flushes to the
 * sentinelPolicies collection by diffing draft vs server and emitting
 * per-policy insert/update/delete; Discard drops the draft.
 *
 * Add and Delete intentionally bypass the draft — they're handled at the
 * parent against the collection directly.
 */
export function useSentinelDraft({ envAId, envBId, policiesA, policiesB }: Args) {
  const [draft, setDraft] = useState<MergedPolicy[] | null>(null);

  const serverMerged = useMemo(() => mergePolicies(policiesA, policiesB), [policiesA, policiesB]);

  const merged = draft ?? serverMerged;

  const hasPending = useMemo(() => {
    if (draft === null) {
      return false;
    }
    return (
      !policiesEqual(toEnv(draft, "envA"), policiesA) ||
      !policiesEqual(toEnv(draft, "envB"), policiesB)
    );
  }, [draft, policiesA, policiesB]);

  const edit = useCallback(
    (fn: (prev: MergedPolicy[]) => MergedPolicy[]) => {
      setDraft((prev) => fn(prev ?? serverMerged));
    },
    [serverMerged],
  );

  const actions = useMemo<SentinelDraftActions>(
    () => ({
      reorder: (next) => edit(() => next),
      toggleEnv: (id, env) =>
        edit((prev) =>
          prev.map((m) => {
            if (m.id !== id || m[env] === null) {
              return m;
            }
            return { ...m, [env]: { ...m[env], enabled: !m[env].enabled } };
          }),
        ),
      addToEnv: (id, env) => {
        const other: "envA" | "envB" = env === "envA" ? "envB" : "envA";
        edit((prev) =>
          prev.map((m) => {
            if (m.id !== id || m[other] === null) {
              return m;
            }
            return { ...m, [env]: { ...m[other], enabled: false } };
          }),
        );
      },
      saveConfig: (id, prodPolicy, previewPolicy) =>
        edit((prev) =>
          prev.map((m) => {
            if (m.id !== id) {
              return m;
            }
            return {
              ...m,
              name: prodPolicy.name,
              envA: m.envA !== null ? prodPolicy : null,
              envB: m.envB !== null && previewPolicy !== null ? previewPolicy : m.envB,
            };
          }),
        ),
    }),
    [edit],
  );

  const save = useCallback(() => {
    if (draft === null) {
      return;
    }
    flushDiff(envAId, policiesA, toEnv(draft, "envA"));
    flushDiff(envBId, policiesB, toEnv(draft, "envB"));
    setDraft(null);
  }, [draft, envAId, envBId, policiesA, policiesB]);

  const discard = useCallback(() => setDraft(null), []);

  return { merged, hasPending, actions, save, discard };
}

export type MergedPolicy = {
  id: string;
  name: string;
  type: SentinelPolicy["type"];
  envA: SentinelPolicy | null;
  envB: SentinelPolicy | null;
};

export function mergePolicies(
  policiesA: SentinelPolicy[],
  policiesB: SentinelPolicy[],
): MergedPolicy[] {
  const mapB = new Map(policiesB.map((p) => [p.id, p]));
  const mapA = new Map(policiesA.map((p) => [p.id, p]));

  const result: MergedPolicy[] = policiesA.map((p) => ({
    id: p.id,
    name: p.name,
    type: p.type,
    envA: p,
    envB: mapB.get(p.id) ?? null,
  }));
  for (const p of policiesB) {
    if (!mapA.has(p.id)) {
      result.push({ id: p.id, name: p.name, type: p.type, envA: null, envB: p });
    }
  }
  return result;
}

const toEnv = (merged: MergedPolicy[], env: "envA" | "envB"): SentinelPolicy[] =>
  merged.flatMap((m) => (m[env] !== null ? [m[env]] : []));

function policiesEqual(a: SentinelPolicy[], b: SentinelPolicy[]): boolean {
  if (a.length !== b.length) {
    return false;
  }
  for (let i = 0; i < a.length; i++) {
    if (JSON.stringify(a[i]) !== JSON.stringify(b[i])) {
      return false;
    }
  }
  return true;
}

/**
 * Apply a diff between server policies and target policies for one
 * environment by emitting per-policy insert/update/delete on the
 * sentinelPolicies collection. Reorder-only changes (same set of ids,
 * different order) are skipped — the underlying blob preserves order, but
 * since each policy id is its own row there's no per-row "position" field
 * to update; full reorder requires rewriting the blob and is intentionally
 * out of scope for this refactor.
 */
function flushDiff(
  environmentId: string,
  server: SentinelPolicy[],
  target: SentinelPolicy[],
): void {
  if (!environmentId) {
    return;
  }
  const serverMap = new Map(server.map((p) => [p.id, p]));
  const targetMap = new Map(target.map((p) => [p.id, p]));

  // Inserts and updates.
  for (const next of target) {
    const prev = serverMap.get(next.id);
    if (!prev) {
      collection.sentinelPolicies.insert({ ...next, environmentId });
      continue;
    }
    if (JSON.stringify(prev) !== JSON.stringify(next)) {
      const key = `${environmentId}::${next.id}`;
      collection.sentinelPolicies.update(key, (draft) => {
        Object.assign(draft, next);
      });
    }
  }

  // Deletes.
  for (const prev of server) {
    if (!targetMap.has(prev.id)) {
      collection.sentinelPolicies.delete(`${environmentId}::${prev.id}`);
    }
  }
}

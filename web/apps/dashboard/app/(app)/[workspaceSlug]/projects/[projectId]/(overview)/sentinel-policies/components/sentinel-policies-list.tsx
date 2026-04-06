"use client";

import { collection } from "@/lib/collections";
import type { SentinelPolicy } from "@/lib/trpc/routers/deploy/environment-settings/sentinel/update-middleware";
import { useCallback, useEffect, useState } from "react";
import { SentinelPolicyRow } from "./sentinel-policy-row";

type MergedPolicy = {
  id: string;
  name: string;
  type: SentinelPolicy["type"];
  /** Full policy object for envA (production), or null if not present in that env. */
  envA: SentinelPolicy | null;
  /** Full policy object for envB (preview), or null if not present in that env. */
  envB: SentinelPolicy | null;
};

function mergePolicies(policiesA: SentinelPolicy[], policiesB: SentinelPolicy[]): MergedPolicy[] {
  const mapA = new Map(policiesA.map((p) => [p.id, p]));
  const mapB = new Map(policiesB.map((p) => [p.id, p]));

  // EnvA order is canonical; envB-only policies append at the end
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

function toEnvPolicies(merged: MergedPolicy[], env: "envA" | "envB"): SentinelPolicy[] {
  return merged.flatMap((m) => (m[env] !== null ? [m[env]] : []));
}

type SentinelPoliciesListProps = {
  envAId: string;
  envBId: string;
  envASlug: string;
  envBSlug: string;
  policiesA: SentinelPolicy[];
  policiesB: SentinelPolicy[];
  topOffset?: number;
};

export function SentinelPoliciesList({
  envAId,
  envBId,
  envASlug,
  envBSlug,
  policiesA,
  policiesB,
  topOffset: _topOffset,
}: SentinelPoliciesListProps) {
  const [mergedPolicies, setMergedPolicies] = useState(() => mergePolicies(policiesA, policiesB));
  const [dragSrcIndex, setDragSrcIndex] = useState<number | null>(null);
  const [dragOverIndex, setDragOverIndex] = useState<number | null>(null);

  useEffect(() => {
    setMergedPolicies((prev) => {
      const prevIds = new Set(prev.map((m) => m.id));
      const incomingIds = new Set([...policiesA.map((p) => p.id), ...policiesB.map((p) => p.id)]);
      const sameSet =
        prevIds.size === incomingIds.size && [...prevIds].every((id) => incomingIds.has(id));

      if (sameSet) {
        // Same policy set — preserve user's drag order, just sync field values
        const mapA = new Map(policiesA.map((p) => [p.id, p]));
        const mapB = new Map(policiesB.map((p) => [p.id, p]));
        return prev.map((m) => ({
          ...m,
          envA: mapA.get(m.id) ?? null,
          envB: mapB.get(m.id) ?? null,
        }));
      }

      // Policy set changed — re-derive from scratch
      return mergePolicies(policiesA, policiesB);
    });
  }, [policiesA, policiesB]);

  const persistBoth = useCallback(
    (updated: MergedPolicy[]) => {
      collection.environmentSettings.update(envAId, (draft) => {
        draft.sentinelConfig = { policies: toEnvPolicies(updated, "envA") };
      });
      collection.environmentSettings.update(envBId, (draft) => {
        draft.sentinelConfig = { policies: toEnvPolicies(updated, "envB") };
      });
    },
    [envAId, envBId],
  );

  const persistEnv = useCallback((updated: MergedPolicy[], envId: string, env: "envA" | "envB") => {
    collection.environmentSettings.update(envId, (draft) => {
      draft.sentinelConfig = { policies: toEnvPolicies(updated, env) };
    });
  }, []);

  const handleReorder = useCallback(
    (newOrder: MergedPolicy[]) => {
      setMergedPolicies(newOrder);
      persistBoth(newOrder);
    },
    [persistBoth],
  );

  const handleToggleEnvA = useCallback(
    (id: string) => {
      setMergedPolicies((prev) => {
        const next = prev.map((m) =>
          m.id === id && m.envA !== null
            ? { ...m, envA: { ...m.envA, enabled: !m.envA.enabled } }
            : m,
        );
        persistEnv(next, envAId, "envA");
        return next;
      });
    },
    [persistEnv, envAId],
  );

  const handleToggleEnvB = useCallback(
    (id: string) => {
      setMergedPolicies((prev) => {
        const next = prev.map((m) =>
          m.id === id && m.envB !== null
            ? { ...m, envB: { ...m.envB, enabled: !m.envB.enabled } }
            : m,
        );
        persistEnv(next, envBId, "envB");
        return next;
      });
    },
    [persistEnv, envBId],
  );

  const handleAddToEnvA = useCallback(
    (id: string) => {
      setMergedPolicies((prev) => {
        const policy = prev.find((m) => m.id === id);
        if (!policy) return prev;
        const base = policy.envB;
        if (!base) return prev;
        const added: SentinelPolicy = { ...base, enabled: false };
        const next = prev.map((m) => (m.id === id ? { ...m, envA: added } : m));
        persistEnv(next, envAId, "envA");
        return next;
      });
    },
    [persistEnv, envAId],
  );

  const handleAddToEnvB = useCallback(
    (id: string) => {
      setMergedPolicies((prev) => {
        const policy = prev.find((m) => m.id === id);
        if (!policy) return prev;
        const base = policy.envA;
        if (!base) return prev;
        const added: SentinelPolicy = { ...base, enabled: false };
        const next = prev.map((m) => (m.id === id ? { ...m, envB: added } : m));
        persistEnv(next, envBId, "envB");
        return next;
      });
    },
    [persistEnv, envBId],
  );

  const handleSaveConfig = useCallback(
    (id: string, prodPolicy: SentinelPolicy, previewPolicy: SentinelPolicy | null) => {
      setMergedPolicies((prev) => {
        const next = prev.map((m) => {
          if (m.id !== id) return m;
          return {
            ...m,
            name: prodPolicy.name,
            envA: m.envA !== null ? prodPolicy : null,
            envB: m.envB !== null && previewPolicy !== null ? previewPolicy : m.envB,
          };
        });
        persistBoth(next);
        return next;
      });
    },
    [persistBoth],
  );

  const handleDelete = useCallback(
    (id: string) => {
      setMergedPolicies((prev) => {
        const next = prev.filter((m) => m.id !== id);
        persistBoth(next);
        return next;
      });
    },
    [persistBoth],
  );

  const handleDragStart = useCallback((index: number) => {
    setDragSrcIndex(index);
  }, []);

  const handleDragOver = useCallback((index: number) => {
    setDragOverIndex(index);
  }, []);

  const handleDrop = useCallback(
    (targetIndex: number) => {
      if (dragSrcIndex === null || dragSrcIndex === targetIndex) {
        setDragSrcIndex(null);
        setDragOverIndex(null);
        return;
      }
      const next = [...mergedPolicies];
      const [item] = next.splice(dragSrcIndex, 1);
      next.splice(targetIndex, 0, item);
      handleReorder(next);
      setDragSrcIndex(null);
      setDragOverIndex(null);
    },
    [dragSrcIndex, mergedPolicies, handleReorder],
  );

  const handleDragEnd = useCallback(() => {
    setDragSrcIndex(null);
    setDragOverIndex(null);
  }, []);

  return (
    <div className="border border-grayA-4 rounded-[14px] overflow-hidden">
      <div>
        {mergedPolicies.map((policy, i) => (
          <SentinelPolicyRow
            key={policy.id}
            policy={policy}
            index={i}
            isLast={i === mergedPolicies.length - 1}
            isDragOver={dragOverIndex === i}
            envASlug={envASlug}
            envBSlug={envBSlug}
            onToggleEnvA={handleToggleEnvA}
            onToggleEnvB={handleToggleEnvB}
            onAddToEnvA={handleAddToEnvA}
            onAddToEnvB={handleAddToEnvB}
            onSave={handleSaveConfig}
            onDelete={handleDelete}
            onDragStart={handleDragStart}
            onDragOver={handleDragOver}
            onDrop={handleDrop}
            onDragEnd={handleDragEnd}
          />
        ))}
      </div>
    </div>
  );
}

"use client";

import { collection } from "@/lib/collections";
import type { SentinelPolicy } from "@/lib/trpc/routers/deploy/environment-settings/sentinel/update-middleware";
import { useCallback, useEffect, useState } from "react";

/** Stable partition: active policies first, inactive after. Relative order within each group preserved. */
function sortByActive(policies: SentinelPolicy[]): SentinelPolicy[] {
  return [...policies.filter((p) => p.enabled), ...policies.filter((p) => !p.enabled)];
}
import { SentinelPolicyRow } from "./sentinel-policy-row";

type SentinelPoliciesListProps = {
  environmentId: string;
  policies: SentinelPolicy[];
};

export function SentinelPoliciesList({ environmentId, policies }: SentinelPoliciesListProps) {
  const [orderedPolicies, setOrderedPolicies] = useState(() => sortByActive(policies));
  const [dragSrcIndex, setDragSrcIndex] = useState<number | null>(null);
  const [dragOverIndex, setDragOverIndex] = useState<number | null>(null);
  useEffect(() => {
    setOrderedPolicies((prev) => {
      const prevIds = new Set(prev.map((p) => p.id));
      const newIds = new Set(policies.map((p) => p.id));
      const sameSet = prevIds.size === newIds.size && [...prevIds].every((id) => newIds.has(id));

      if (sameSet) {
        // Same policy set — preserve user's order, just sync updated field values
        return prev.map((p) => policies.find((np) => np.id === p.id) ?? p);
      }
      // Policies added or removed — re-sort from scratch
      return sortByActive(policies);
    });
  }, [policies]);

  const persist = useCallback(
    (updated: SentinelPolicy[]) => {
      collection.environmentSettings.update(environmentId, (draft) => {
        draft.sentinelConfig = { policies: updated };
      });
    },
    [environmentId],
  );

  const handleReorder = useCallback(
    (newOrder: SentinelPolicy[]) => {
      setOrderedPolicies(newOrder);
      persist(newOrder);
    },
    [persist],
  );

  const handleToggleActive = useCallback(
    (id: string) => {
      setOrderedPolicies((prev) => {
        const toggled = prev.map((p) => (p.id === id ? { ...p, enabled: !p.enabled } : p));
        const next = sortByActive(toggled);
        persist(next);
        return next;
      });
    },
    [persist],
  );

  const handleUpdate = useCallback(
    (id: string, field: "name", value: string) => {
      setOrderedPolicies((prev) => {
        const next = prev.map((p) => (p.id === id ? { ...p, [field]: value } : p));
        persist(next);
        return next;
      });
    },
    [persist],
  );

  const handleDelete = useCallback(
    (id: string) => {
      setOrderedPolicies((prev) => {
        const next = prev.filter((p) => p.id !== id);
        persist(next);
        return next;
      });
    },
    [persist],
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
      const next = [...orderedPolicies];
      const [item] = next.splice(dragSrcIndex, 1);
      next.splice(targetIndex, 0, item);
      handleReorder(next);
      setDragSrcIndex(null);
      setDragOverIndex(null);
    },
    [dragSrcIndex, orderedPolicies, handleReorder],
  );

  const handleDragEnd = useCallback(() => {
    setDragSrcIndex(null);
    setDragOverIndex(null);
  }, []);

  return (
    <div className="border border-grayA-4 rounded-[14px] overflow-hidden">
      <div>
        {orderedPolicies.map((policy, i) => (
          <SentinelPolicyRow
            key={policy.id}
            policy={policy}
            index={i}
            isLast={i === orderedPolicies.length - 1}
            isDragOver={dragOverIndex === i}
            onToggleActive={handleToggleActive}
            onUpdate={handleUpdate}
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

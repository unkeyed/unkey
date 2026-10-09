"use client";

import { ResourceListBody } from "@unkey/ui";
import { useCallback, useState } from "react";
import type { PolicySwitches } from "../../hooks/policy-switches";
import type { MergedPolicy, PolicyEnvs, PolicyRowKey } from "./merge";
import { PoliciesListHeader, PolicyRow } from "./row";

type PoliciesListProps = {
  envs: PolicyEnvs;
  merged: MergedPolicy[];
  switchesOf: (policy: MergedPolicy) => PolicySwitches;
  onReorder: (fromKey: PolicyRowKey, toKey: PolicyRowKey) => void;
  onDelete: (key: PolicyRowKey) => void;
  onEdit: (key: PolicyRowKey) => void;
};

export function PoliciesList({
  envs,
  merged,
  switchesOf,
  onReorder,
  onDelete,
  onEdit,
}: PoliciesListProps) {
  const [dragSrcIndex, setDragSrcIndex] = useState<number | null>(null);
  const [dragOverIndex, setDragOverIndex] = useState<number | null>(null);

  const resetDrag = useCallback(() => {
    setDragSrcIndex(null);
    setDragOverIndex(null);
  }, []);

  const handleDragStart = useCallback((index: number) => {
    setDragSrcIndex(index);
  }, []);

  const handleDragOver = useCallback((index: number) => {
    setDragOverIndex(index);
  }, []);

  const handleDrop = useCallback(
    (targetIndex: number) => {
      const from = dragSrcIndex === null ? undefined : merged[dragSrcIndex];
      const to = merged[targetIndex];
      if (from && to && from !== to) {
        onReorder(from.key, to.key);
      }
      resetDrag();
    },
    [dragSrcIndex, merged, onReorder, resetDrag],
  );

  return (
    <div>
      <PoliciesListHeader />
      <ResourceListBody>
        {merged.map((policy, i) => (
          <PolicyRow
            key={policy.key}
            policy={policy}
            switches={switchesOf(policy)}
            index={i}
            envs={envs}
            isDragOver={dragOverIndex === i}
            onDelete={onDelete}
            onEdit={onEdit}
            onDragStart={handleDragStart}
            onDragOver={handleDragOver}
            onDrop={handleDrop}
            onDragEnd={resetDrag}
          />
        ))}
      </ResourceListBody>
    </div>
  );
}

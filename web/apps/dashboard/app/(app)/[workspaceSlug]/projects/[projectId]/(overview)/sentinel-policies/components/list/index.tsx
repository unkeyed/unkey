"use client";

import type { SentinelPolicy } from "@/lib/trpc/routers/deploy/environment-settings/sentinel/update-middleware";
import { useCallback, useState } from "react";
import { SentinelPolicyRow } from "./row";
import type { MergedPolicy, SentinelDraftActions } from "./use-sentinel-draft";

type SentinelPoliciesListProps = {
  envASlug: string;
  envBSlug: string;
  merged: MergedPolicy[];
  actions: SentinelDraftActions;
  onDelete: (id: string) => void;
};

export function SentinelPoliciesList({
  envASlug,
  envBSlug,
  merged,
  actions,
  onDelete,
}: SentinelPoliciesListProps) {
  const [dragSrcIndex, setDragSrcIndex] = useState<number | null>(null);
  const [dragOverIndex, setDragOverIndex] = useState<number | null>(null);

  const handleToggleEnvA = useCallback((id: string) => actions.toggleEnv(id, "envA"), [actions]);
  const handleToggleEnvB = useCallback((id: string) => actions.toggleEnv(id, "envB"), [actions]);
  const handleAddToEnvA = useCallback((id: string) => actions.addToEnv(id, "envA"), [actions]);
  const handleAddToEnvB = useCallback((id: string) => actions.addToEnv(id, "envB"), [actions]);
  const handleSaveConfig = useCallback(
    (id: string, prodPolicy: SentinelPolicy, previewPolicy: SentinelPolicy | null) =>
      actions.saveConfig(id, prodPolicy, previewPolicy),
    [actions],
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
      const next = [...merged];
      const [item] = next.splice(dragSrcIndex, 1);
      next.splice(targetIndex, 0, item);
      actions.reorder(next);
      setDragSrcIndex(null);
      setDragOverIndex(null);
    },
    [dragSrcIndex, merged, actions],
  );

  const handleDragEnd = useCallback(() => {
    setDragSrcIndex(null);
    setDragOverIndex(null);
  }, []);

  return (
    <div className="border border-grayA-4 rounded-[14px] overflow-hidden">
      <div>
        {merged.map((policy, i) => (
          <SentinelPolicyRow
            key={policy.id}
            policy={policy}
            index={i}
            isLast={i === merged.length - 1}
            isDragOver={dragOverIndex === i}
            envASlug={envASlug}
            envBSlug={envBSlug}
            onToggleEnvA={handleToggleEnvA}
            onToggleEnvB={handleToggleEnvB}
            onAddToEnvA={handleAddToEnvA}
            onAddToEnvB={handleAddToEnvB}
            onSave={handleSaveConfig}
            onDelete={onDelete}
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

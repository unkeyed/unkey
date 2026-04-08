"use client";

import type { SentinelPolicy } from "@/lib/collections/deploy/sentinel-policies.schema";
import { useCallback, useState } from "react";
import type { MergedPolicy } from "./merge";
import { SentinelPolicyRow } from "./row";

type SentinelPoliciesListProps = {
  envASlug: string;
  envBSlug: string;
  merged: MergedPolicy[];
  onToggleEnv: (id: string, env: "envA" | "envB") => void;
  onAddToEnv: (id: string, env: "envA" | "envB") => void;
  onReorder: (envs: ("envA" | "envB")[], orderedIds: string[]) => void;
  onDelete: (id: string) => void;
  onEdit: (policy: SentinelPolicy) => void;
};

export function SentinelPoliciesList({
  envASlug,
  envBSlug,
  merged,
  onToggleEnv,
  onAddToEnv,
  onReorder,
  onDelete,
  onEdit,
}: SentinelPoliciesListProps) {
  const [dragSrcIndex, setDragSrcIndex] = useState<number | null>(null);
  const [dragOverIndex, setDragOverIndex] = useState<number | null>(null);

  const handleToggleEnvA = useCallback((id: string) => onToggleEnv(id, "envA"), [onToggleEnv]);
  const handleToggleEnvB = useCallback((id: string) => onToggleEnv(id, "envB"), [onToggleEnv]);
  const handleAddToEnvA = useCallback((id: string) => onAddToEnv(id, "envA"), [onAddToEnv]);
  const handleAddToEnvB = useCallback((id: string) => onAddToEnv(id, "envB"), [onAddToEnv]);

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
      // Emit a reorder for whichever envs the dragged row exists in. The
      // server is tolerant — extra/missing ids are reconciled — so we send
      // the full merged-id sequence to each env independently.
      const orderedIds = next.map((m) => m.id);
      const envs: ("envA" | "envB")[] = [];
      if (item.envA !== null) envs.push("envA");
      if (item.envB !== null) envs.push("envB");
      if (envs.length > 0) onReorder(envs, orderedIds);
      setDragSrcIndex(null);
      setDragOverIndex(null);
    },
    [dragSrcIndex, merged, onReorder],
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
            onDelete={onDelete}
            onEdit={onEdit}
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

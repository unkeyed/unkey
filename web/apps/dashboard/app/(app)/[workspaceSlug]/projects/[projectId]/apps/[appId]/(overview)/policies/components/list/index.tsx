"use client";

import type { PolicyRow as PolicyRowData } from "@/lib/collections/deploy/policies";
import type { Policy } from "@/lib/collections/deploy/policies.schema";
import { useCallback, useState } from "react";
import type { MergedPolicy } from "./merge";
import { PolicyRow } from "./row";

type PoliciesListProps = {
  envASlug: string;
  envBSlug: string;
  merged: MergedPolicy[];
  onToggleEnv: (key: string, env: "envA" | "envB") => void;
  onAddToEnv: (key: string, env: "envA" | "envB") => void;
  onReorder: (envs: ("envA" | "envB")[], rowsByEnv: Record<string, PolicyRowData[]>) => void;
  onDelete: (key: string) => void;
  onEdit: (policy: Policy) => void;
};

export function PoliciesList({
  envASlug,
  envBSlug,
  merged,
  onToggleEnv,
  onAddToEnv,
  onReorder,
  onDelete,
  onEdit,
}: PoliciesListProps) {
  const [dragSrcIndex, setDragSrcIndex] = useState<number | null>(null);
  const [dragOverIndex, setDragOverIndex] = useState<number | null>(null);

  const handleToggleEnvA = useCallback((key: string) => onToggleEnv(key, "envA"), [onToggleEnv]);
  const handleToggleEnvB = useCallback((key: string) => onToggleEnv(key, "envB"), [onToggleEnv]);
  const handleAddToEnvA = useCallback((key: string) => onAddToEnv(key, "envA"), [onAddToEnv]);
  const handleAddToEnvB = useCallback((key: string) => onAddToEnv(key, "envB"), [onAddToEnv]);

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
      const envs: ("envA" | "envB")[] = [];
      if (item.envA !== null) {
        envs.push("envA");
      }
      if (item.envB !== null) {
        envs.push("envB");
      }
      if (envs.length > 0) {
        // `next` is already in the wanted order. Send the rows, so no later
        // code must match a policy by name or id.
        const rowsByEnv: Record<string, PolicyRowData[]> = {};
        for (const env of envs) {
          rowsByEnv[env] = next
            .map((m) => m[env])
            .filter((row): row is PolicyRowData => row !== null);
        }
        onReorder(envs, rowsByEnv);
      }
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
    <div className="border border-grayA-4 rounded-lg overflow-hidden">
      {merged.map((policy, i) => (
        <PolicyRow
          key={policy.key}
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
  );
}

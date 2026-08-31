"use client";

import type { PolicyRow as PolicyRowData } from "@/lib/collections/deploy/policies";
import type { Policy } from "@/lib/collections/deploy/policies.schema";
import { useCallback, useState } from "react";
import type { Env, MergedPolicy } from "./merge";
import { PolicyRow } from "./row";

type PoliciesListProps = {
  productionSlug: string;
  previewSlug: string;
  merged: MergedPolicy[];
  onToggleEnv: (key: string, env: Env) => void;
  onAddToEnv: (key: string, env: Env) => void;
  onReorder: (envs: Env[], rowsByEnv: Partial<Record<Env, PolicyRowData[]>>) => void;
  onDelete: (key: string) => void;
  onEdit: (policy: Policy) => void;
};

export function PoliciesList({
  productionSlug,
  previewSlug,
  merged,
  onToggleEnv,
  onAddToEnv,
  onReorder,
  onDelete,
  onEdit,
}: PoliciesListProps) {
  const [dragSrcIndex, setDragSrcIndex] = useState<number | null>(null);
  const [dragOverIndex, setDragOverIndex] = useState<number | null>(null);

  const handleToggleProduction = useCallback(
    (key: string) => onToggleEnv(key, "production"),
    [onToggleEnv],
  );
  const handleTogglePreview = useCallback(
    (key: string) => onToggleEnv(key, "preview"),
    [onToggleEnv],
  );
  const handleAddToProduction = useCallback(
    (key: string) => onAddToEnv(key, "production"),
    [onAddToEnv],
  );
  const handleAddToPreview = useCallback((key: string) => onAddToEnv(key, "preview"), [onAddToEnv]);

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
      const envs: Env[] = [];
      if (item.production !== null) {
        envs.push("production");
      }
      if (item.preview !== null) {
        envs.push("preview");
      }
      if (envs.length > 0) {
        // `next` is already in the wanted order. Send the rows, so no later
        // code must match a policy by name or id.
        const rowsByEnv: Partial<Record<Env, PolicyRowData[]>> = {};
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
          productionSlug={productionSlug}
          previewSlug={previewSlug}
          onToggleProduction={handleToggleProduction}
          onTogglePreview={handleTogglePreview}
          onAddToProduction={handleAddToProduction}
          onAddToPreview={handleAddToPreview}
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

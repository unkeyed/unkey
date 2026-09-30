"use client";

import type { PolicyRow as PolicyRowData } from "@/lib/collections/deploy/policies";
import type { Policy } from "@/lib/collections/deploy/policies.schema";
import { useCallback, useState } from "react";
import { PolicyRow } from "./row";

type PoliciesListProps = {
  rows: PolicyRowData[];
  onToggle: (id: string) => void;
  onReorder: (rows: PolicyRowData[]) => void;
  onDelete: (id: string) => void;
  onEdit: (policy: Policy) => void;
};

export function PoliciesList({ rows, onToggle, onReorder, onDelete, onEdit }: PoliciesListProps) {
  const [dragSrcIndex, setDragSrcIndex] = useState<number | null>(null);
  const [dragOverIndex, setDragOverIndex] = useState<number | null>(null);

  const handleDragStart = useCallback((index: number) => {
    setDragSrcIndex(index);
  }, []);

  const handleDragOver = useCallback((index: number) => {
    setDragOverIndex(index);
  }, []);

  const handleDrop = useCallback(
    (targetIndex: number) => {
      if (dragSrcIndex !== null && dragSrcIndex !== targetIndex) {
        const next = [...rows];
        const [item] = next.splice(dragSrcIndex, 1);
        next.splice(targetIndex, 0, item);
        onReorder(next);
      }
      setDragSrcIndex(null);
      setDragOverIndex(null);
    },
    [dragSrcIndex, rows, onReorder],
  );

  const handleDragEnd = useCallback(() => {
    setDragSrcIndex(null);
    setDragOverIndex(null);
  }, []);

  return (
    <div className="border bg-raised rounded-lg overflow-hidden">
      {rows.map((policy, i) => (
        <PolicyRow
          key={policy.id}
          policy={policy}
          index={i}
          isLast={i === rows.length - 1}
          isDragOver={dragOverIndex === i}
          onToggle={onToggle}
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

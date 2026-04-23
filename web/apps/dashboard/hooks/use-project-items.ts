"use client";

import {
  PROJECT_ITEMS_EVENT,
  type ProjectItem,
  type ProjectItemType,
  addProjectItem as addItemLib,
  getProjectItems,
} from "@/lib/project-items";
import { useCallback, useEffect, useMemo, useState } from "react";

/**
 * Prototype-only hook backing the project-leaf sidebar and overview page.
 * Mirrors the shape of `use-product-selection.ts` — localStorage + a
 * custom window event for cross-tab sync.
 */
export function useProjectItems(projectId: string) {
  const [items, setItems] = useState<ProjectItem[]>(() =>
    typeof window === "undefined" ? [] : getProjectItems(projectId),
  );

  useEffect(() => {
    setItems(getProjectItems(projectId));

    const onChange = () => setItems(getProjectItems(projectId));
    window.addEventListener(PROJECT_ITEMS_EVENT, onChange);
    window.addEventListener("storage", onChange);
    return () => {
      window.removeEventListener(PROJECT_ITEMS_EVENT, onChange);
      window.removeEventListener("storage", onChange);
    };
  }, [projectId]);

  const addItem = useCallback(
    (input: { type: ProjectItemType; name: string }) => addItemLib(projectId, input),
    [projectId],
  );

  return useMemo(() => ({ items, addItem }), [items, addItem]);
}

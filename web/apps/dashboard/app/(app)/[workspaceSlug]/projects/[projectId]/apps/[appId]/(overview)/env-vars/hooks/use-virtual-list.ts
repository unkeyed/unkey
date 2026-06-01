import { useVirtualizer } from "@tanstack/react-virtual";
import { useCallback, useEffect, useRef, useState } from "react";
import type { DisplayRow } from "../components/list/env-var-item-row";

export function useVirtualList(
  displayRows: DisplayRow[],
  editingId: string | null,
  expandedRow: string | null,
) {
  const listRef = useRef<HTMLDivElement>(null);
  const [scrollElement, setScrollElement] = useState<HTMLElement | null>(null);
  const [scrollMargin, setScrollMargin] = useState(0);

  const listRefCallback = useCallback((node: HTMLDivElement | null) => {
    listRef.current = node;
    if (!node) {
      return;
    }
    let el: HTMLElement | null = node.parentElement;
    while (el) {
      const { overflow, overflowY } = getComputedStyle(el);
      if (
        overflow === "auto" ||
        overflow === "scroll" ||
        overflowY === "auto" ||
        overflowY === "scroll"
      ) {
        setScrollElement(el);
        setScrollMargin(
          node.getBoundingClientRect().top - el.getBoundingClientRect().top + el.scrollTop,
        );
        break;
      }
      el = el.parentElement;
    }
  }, []);

  const getItemKey = useCallback(
    (index: number) => {
      const row = displayRows[index];
      return row.kind === "single" ? row.item.id : `group-${row.key}`;
    },
    [displayRows],
  );

  const ROW_HEIGHT = 70;
  const EDIT_HEIGHT = 470;

  // Pre-calculate row heights so the virtualizer positions items correctly
  // before ResizeObserver measures the DOM. Without this, there's a 1-frame
  // delay where items below an expanding row keep their old positions,
  // causing a visible flash/ghost artifact.
  const estimateSize = useCallback(
    (index: number) => {
      const row = displayRows[index];

      if (row.kind === "single") {
        return row.item.id === editingId ? ROW_HEIGHT + EDIT_HEIGHT : ROW_HEIGHT;
      }

      if (expandedRow !== row.key) {
        return ROW_HEIGHT;
      }

      // Expanded group: header + each sub-item (+ edit form if editing inside)
      let height = ROW_HEIGHT + row.items.length * ROW_HEIGHT;
      if (editingId && row.items.some((item) => item.id === editingId)) {
        height += EDIT_HEIGHT;
      }
      return height;
    },
    [displayRows, editingId, expandedRow],
  );

  const virtualizer = useVirtualizer({
    count: displayRows.length,
    getScrollElement: useCallback(() => scrollElement, [scrollElement]),
    estimateSize,
    overscan: 5,
    getItemKey,
    scrollMargin,
  });

  // biome-ignore lint/correctness/useExhaustiveDependencies: editingId/expandedGroups trigger measurement invalidation so estimateSize is re-consulted
  useEffect(() => {
    virtualizer.measure();
  }, [editingId, expandedRow, virtualizer]);

  return { virtualizer, listRefCallback, scrollMargin };
}

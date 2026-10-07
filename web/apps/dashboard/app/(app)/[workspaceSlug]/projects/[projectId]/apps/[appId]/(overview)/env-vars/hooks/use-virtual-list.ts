import { useVirtualizer } from "@tanstack/react-virtual";
import { useCallback, useEffect, useState } from "react";

export function useVirtualList(rows: { id: string }[], editingId: string | null) {
  const [scrollElement, setScrollElement] = useState<HTMLElement | null>(null);
  const [scrollMargin, setScrollMargin] = useState(0);

  const listRefCallback = useCallback((node: HTMLDivElement | null) => {
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

  const getItemKey = useCallback((index: number) => rows[index].id, [rows]);

  const ROW_HEIGHT = 48;
  const EDIT_HEIGHT = 470;

  // Pre-calculate row heights so the virtualizer positions items correctly
  // before ResizeObserver measures the DOM. Without this, there's a 1-frame
  // delay where items below an expanding row keep their old positions,
  // causing a visible flash/ghost artifact.
  const estimateSize = useCallback(
    (index: number) => (rows[index].id === editingId ? ROW_HEIGHT + EDIT_HEIGHT : ROW_HEIGHT),
    [rows, editingId],
  );

  const virtualizer = useVirtualizer({
    count: rows.length,
    getScrollElement: useCallback(() => scrollElement, [scrollElement]),
    estimateSize,
    overscan: 5,
    getItemKey,
    scrollMargin,
  });

  // biome-ignore lint/correctness/useExhaustiveDependencies: editingId triggers measurement invalidation so estimateSize is re-consulted
  useEffect(() => {
    virtualizer.measure();
  }, [editingId, virtualizer]);

  return { virtualizer, listRefCallback, scrollMargin };
}

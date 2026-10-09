import { useVirtualizer } from "@tanstack/react-virtual";
import { findScrollParent } from "@unkey/ui";
import { useCallback, useState } from "react";

const ROW_HEIGHT = 48;

export function useVirtualList(rows: { id: string }[]) {
  const [scrollElement, setScrollElement] = useState<HTMLElement | null>(null);
  const [scrollMargin, setScrollMargin] = useState(0);

  const listRefCallback = useCallback((node: HTMLDivElement | null) => {
    if (!node) {
      return;
    }
    const el = findScrollParent(node);
    if (!el) {
      return;
    }
    setScrollElement(el);
    setScrollMargin(
      node.getBoundingClientRect().top - el.getBoundingClientRect().top + el.scrollTop,
    );
  }, []);

  const getItemKey = useCallback((index: number) => rows[index].id, [rows]);

  const virtualizer = useVirtualizer({
    count: rows.length,
    getScrollElement: useCallback(() => scrollElement, [scrollElement]),
    estimateSize: () => ROW_HEIGHT,
    overscan: 5,
    getItemKey,
    scrollMargin,
  });

  return { virtualizer, listRefCallback, scrollMargin };
}

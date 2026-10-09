import { findScrollParent } from "@unkey/ui";
import { useEffect, useRef } from "react";

export type Field = "key" | "value";

const REVEAL_MARGIN_PX = 48;

function revealFooter(element: HTMLElement) {
  const scroller = findScrollParent(element);
  if (!scroller) {
    return;
  }
  const hidden =
    element.getBoundingClientRect().bottom +
    REVEAL_MARGIN_PX -
    scroller.getBoundingClientRect().bottom;
  if (hidden > 0) {
    scroller.scrollBy({ top: hidden });
  }
}

export function useCellFocus(rowCount: number) {
  const cells = useRef(new Map<string, HTMLElement>());
  const footerRef = useRef<HTMLDivElement>(null);
  const rowCountRef = useRef(rowCount);

  useEffect(() => {
    if (rowCount > rowCountRef.current && footerRef.current) {
      revealFooter(footerRef.current);
    }
    rowCountRef.current = rowCount;
  }, [rowCount]);

  const cellRef = (id: string, field: Field) => (element: HTMLElement | null) => {
    if (!element) {
      return;
    }
    const cellKey = `${id}:${field}`;
    cells.current.set(cellKey, element);
    return () => {
      cells.current.delete(cellKey);
    };
  };

  const focusCell = (id: string, field: Field) => {
    requestAnimationFrame(() => {
      cells.current.get(`${id}:${field}`)?.focus();
    });
  };

  return { cellRef, focusCell, footerRef };
}

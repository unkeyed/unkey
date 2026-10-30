"use client";
import { type CSSProperties, useLayoutEffect, useRef, useState } from "react";
import { STICKY_STACK_H } from "../../page/sticky-stack";

// A sideways scroll box becomes the header's sticky container, so the page
// offset would push the header down over the first rows. Pin only while it fits.
export function usePinnedTableHeader() {
  const boxRef = useRef<HTMLDivElement>(null);
  const tableRef = useRef<HTMLTableElement>(null);
  const [headerPinned, setHeaderPinned] = useState(true);

  useLayoutEffect(() => {
    const box = boxRef.current;
    const table = tableRef.current;
    if (!box || !table) {
      return;
    }
    const measure = () => {
      const { paddingLeft, paddingRight } = getComputedStyle(box);
      const boxContentWidth =
        box.clientWidth - Number.parseFloat(paddingLeft) - Number.parseFloat(paddingRight);
      setHeaderPinned(table.getBoundingClientRect().width <= boxContentWidth + 1);
    };
    measure();
    const observer = new ResizeObserver(measure);
    observer.observe(box);
    observer.observe(table);
    return () => observer.disconnect();
  }, []);

  return {
    boxRef,
    tableRef,
    headerPinned,
    boxClassName: headerPinned ? "overflow-x-clip" : "overflow-x-auto",
    rowsClassName: "[&>tr]:scroll-mt-(--sticky-stack-h)",
    rowsStyle: { "--sticky-stack-h": STICKY_STACK_H } as CSSProperties,
  };
}

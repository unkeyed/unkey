"use client";
import { type VirtualItem, useWindowVirtualizer } from "@tanstack/react-virtual";
import { STICKY_ROOT_SELECTOR } from "@unkey/ui";
import { useLayoutEffect, useState } from "react";

type PageVirtualizerOptions<TItem extends Element> = Omit<
  Parameters<typeof useWindowVirtualizer<TItem>>[0],
  "scrollMargin"
>;

function elementsAbove(element: HTMLElement): Element[] {
  const above: Element[] = [];
  for (
    let node: Element | null = element;
    node && node !== document.body;
    node = node.parentElement
  ) {
    for (
      let sibling = node.previousElementSibling;
      sibling;
      sibling = sibling.previousElementSibling
    ) {
      above.push(sibling);
    }
  }
  return above;
}

function usePageOffset(element: HTMLElement | null): number {
  const [offset, setOffset] = useState(0);
  useLayoutEffect(() => {
    if (!element) {
      return;
    }
    const measure = () => setOffset(element.getBoundingClientRect().top + window.scrollY);
    measure();
    const resizeObserver = new ResizeObserver(measure);
    resizeObserver.observe(document.body);
    for (const above of elementsAbove(element)) {
      resizeObserver.observe(above);
    }
    const stickyRoot = element.closest(STICKY_ROOT_SELECTOR);
    const mutationObserver = new MutationObserver(measure);
    if (stickyRoot) {
      mutationObserver.observe(stickyRoot, { attributeFilter: ["style"] });
    }
    return () => {
      resizeObserver.disconnect();
      mutationObserver.disconnect();
    };
  }, [element]);
  return offset;
}

export function usePageVirtualizer<TItem extends Element>(options: PageVirtualizerOptions<TItem>) {
  const [listElement, listRef] = useState<HTMLElement | null>(null);
  const scrollMargin = usePageOffset(listElement);
  const virtualizer = useWindowVirtualizer<TItem>({ ...options, scrollMargin });
  const items: VirtualItem[] = virtualizer
    .getVirtualItems()
    .map((item) => ({ ...item, start: item.start - scrollMargin, end: item.end - scrollMargin }));
  return { virtualizer, listRef, items };
}

"use client";
import { mergeProps } from "@base-ui/react/merge-props";
import { useRender } from "@base-ui/react/use-render";
import { type RefCallback, useCallback } from "react";
import {
  STICKY_ROOT_SELECTOR,
  type StickySlotName,
  stickySlotHeightVar,
  stickySlotProps,
  warnIfSlotOutOfPlace,
} from "./sticky-stack";

const isDev = typeof process !== "undefined" && process.env.NODE_ENV !== "production";

const slotOwners = new WeakMap<HTMLElement, Map<StickySlotName, HTMLElement>>();

function useStickySlotRef(slot: StickySlotName): RefCallback<HTMLElement> {
  return useCallback(
    (element: HTMLElement | null) => {
      const root = element?.closest<HTMLElement>(STICKY_ROOT_SELECTOR);
      if (!element || !root) {
        return;
      }
      const owners = slotOwners.get(root) ?? new Map<StickySlotName, HTMLElement>();
      slotOwners.set(root, owners);
      if (isDev && owners.has(slot)) {
        console.warn(`Sticky slot "${slot}" is registered twice on one page.`);
      }
      owners.set(slot, element);

      const heightVar = stickySlotHeightVar(slot);
      const writeHeight = (height: number) => root.style.setProperty(heightVar, `${height}px`);
      writeHeight(element.getBoundingClientRect().height);
      if (isDev) {
        warnIfSlotOutOfPlace(slot, element);
      }
      const observer = new ResizeObserver(([entry]) => {
        const height = entry?.borderBoxSize[0]?.blockSize;
        if (owners.get(slot) === element && height !== undefined) {
          writeHeight(height);
        }
      });
      observer.observe(element, { box: "border-box" });

      return () => {
        observer.disconnect();
        if (owners.get(slot) === element) {
          owners.delete(slot);
          root.style.removeProperty(heightVar);
        }
      };
    },
    [slot],
  );
}

type StickySlotProps = useRender.ComponentProps<"div"> & {
  slot: StickySlotName;
  enabled?: boolean;
};

export function StickySlot({ slot, enabled = true, render, ref, ...props }: StickySlotProps) {
  const slotRef = useStickySlotRef(slot);
  return useRender({
    defaultTagName: "div",
    render,
    ref: [enabled ? slotRef : null, ref ?? null],
    props: mergeProps<"div">(enabled ? stickySlotProps(slot) : {}, props),
  });
}

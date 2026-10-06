"use client";

import { TOP_NAV_HEIGHT } from "@/components/navigation/top-nav";
import { cn } from "@unkey/ui/src/lib/utils";
import { type ReactNode, useEffect, useState } from "react";
import { formatStageDuration } from "./panes/deploying/run-model";

export const COLUMN_GAP = "clamp(24px, 4vh, 48px)";
export const COLUMN_TOP_SPACE = "clamp(40px, 8vh, 120px)";
const MIN_BOTTOM_PEEK_PX = 40;
export const COLUMN_CARD_MAX_HEIGHT = `min(640px, calc(100dvh - ${COLUMN_TOP_SPACE} - 2 * ${COLUMN_GAP} - ${TOP_NAV_HEIGHT + MIN_BOTTOM_PEEK_PX}px))`;

export const cardFooter =
  "flex shrink-0 items-center gap-3 border-t border-grayA-4 bg-grayA-2 px-5 py-3";

export const cardSurface = "flex min-h-0 flex-col overflow-hidden rounded-lg border bg-raised";

export function CardTitle({ title, description }: { title: string; description: string }) {
  return (
    <div className="flex flex-col gap-1">
      <h1 className="text-lg font-semibold leading-7 text-gray-12">{title}</h1>
      <p className="text-sm leading-5 text-gray-11">{description}</p>
    </div>
  );
}

export function CardHeader({
  title,
  description,
  elapsedMs,
}: {
  title: string;
  description: string;
  elapsedMs: number | null;
}) {
  return (
    <div className="flex shrink-0 items-start gap-4">
      <CardTitle title={title} description={description} />
      <span className="ml-auto font-mono text-sm tabular-nums text-gray-11">
        {elapsedMs === null ? "" : formatStageDuration(elapsedMs)}
      </span>
    </div>
  );
}

// Rounding and focus rings leave a few pixels of overflow on cards that fit.
const MIN_HIDDEN_PX = 8;

function useScrollFade() {
  const [element, setElement] = useState<HTMLElement | null>(null);
  useEffect(() => {
    if (!element) {
      return;
    }
    const update = () => {
      const below = element.scrollHeight - element.scrollTop - element.clientHeight > MIN_HIDDEN_PX;
      element.toggleAttribute("data-more-below", below);
    };
    update();
    const resize = new ResizeObserver(update);
    resize.observe(element);
    for (const child of element.children) {
      resize.observe(child);
    }
    const mutations = new MutationObserver(update);
    mutations.observe(element, { childList: true, subtree: true });
    element.addEventListener("scroll", update, { passive: true });
    return () => {
      resize.disconnect();
      mutations.disconnect();
      element.removeEventListener("scroll", update);
    };
  }, [element]);
  return setElement;
}

export function CardScrollBody({
  className,
  children,
}: {
  className?: string;
  children: ReactNode;
}) {
  const scrollRef = useScrollFade();
  return (
    <div
      ref={scrollRef}
      className={cn(
        "flex min-h-0 flex-col gap-5 overflow-y-auto overscroll-contain p-5 [scrollbar-width:thin] data-[more-below]:[mask-image:linear-gradient(to_bottom,black_calc(100%-40px),transparent)]",
        className,
      )}
    >
      {children}
    </div>
  );
}

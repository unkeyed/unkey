"use client";

import { cn } from "@unkey/ui/src/lib/utils";
import type { ReactNode } from "react";
import { scrollFadeClass, useScrollFade } from "../../use-scroll-fade";
import { LiveRegionsMap } from "./live-map";
import { formatStageDuration } from "./run-model";

type CongratsRow = { label: string; value: ReactNode };

export function CongratsBody({
  elapsedMs,
  regions,
  rows,
}: { elapsedMs: number | null; regions: string[]; rows: CongratsRow[] }) {
  const scrollRef = useScrollFade();
  return (
    <div
      ref={scrollRef}
      className={cn(
        "flex min-h-0 flex-col gap-5 overflow-y-auto overscroll-contain p-5 [scrollbar-width:thin]",
        scrollFadeClass,
      )}
    >
      <div
        className="flex animate-upgrade-rise flex-col items-center gap-2 text-center motion-reduce:animate-none"
        style={{ animationDelay: "40ms" }}
      >
        <LiveRegionsMap regions={regions} />
        <div className="flex flex-col gap-1">
          <h1 className="font-semibold text-gray-12 text-xl tracking-[-0.03em]">
            Congratulations!
          </h1>
          <p className="text-gray-11 text-sm leading-5">
            {elapsedMs === null
              ? "Your app is live on Unkey."
              : `Your app is live on Unkey. Deployed in ${formatStageDuration(elapsedMs)}.`}
          </p>
        </div>
      </div>
      <div
        className="flex shrink-0 animate-upgrade-rise flex-col overflow-hidden rounded-lg border border-grayA-4 motion-reduce:animate-none"
        style={{ animationDelay: "120ms" }}
      >
        {rows.map((row) => (
          <div
            key={row.label}
            className="flex h-11 items-center gap-3 border-t border-grayA-4 px-4 first:border-t-0"
          >
            <span className="shrink-0 text-sm font-medium text-gray-12">{row.label}</span>
            <span className="flex min-w-0 flex-1 justify-end overflow-hidden text-sm text-gray-11">
              {row.value}
            </span>
          </div>
        ))}
      </div>
    </div>
  );
}

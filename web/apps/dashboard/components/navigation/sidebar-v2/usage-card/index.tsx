"use client";

import { useSidebar } from "@/components/ui/sidebar";
import { cn } from "@/lib/utils";
import { UsagePanel } from "./usage-panel";
import { useUsageSummary } from "./use-usage-summary";

export function UsageCard() {
  const { state } = useSidebar();
  const summary = useUsageSummary();
  const railed = state === "collapsed";

  if (summary === null) {
    return null;
  }

  // `inert`, not `aria-hidden`: the links inside are focusable while hidden.
  return (
    <div
      inert={railed || undefined}
      className={cn(
        "transition-[opacity,transform] duration-200 ease-out motion-reduce:transition-none",
        railed && "-translate-x-1 opacity-0",
      )}
    >
      <UsagePanel summary={summary} />
    </div>
  );
}

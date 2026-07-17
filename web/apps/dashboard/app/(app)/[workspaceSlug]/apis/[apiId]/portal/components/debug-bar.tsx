"use client";

import { cn } from "@/lib/utils";
import type { PortalLifecycleState } from "./use-portal-lifecycle";

const STATES: PortalLifecycleState[] = ["disabled", "enabling", "enabled"];

export function DebugBar({
  state,
  onSelect,
}: {
  state: PortalLifecycleState;
  onSelect: (state: PortalLifecycleState) => void;
}) {
  return (
    <div className="fixed bottom-4 right-4 z-50 flex items-center gap-0.5 rounded-full border border-grayA-6 bg-gray-1 p-1 shadow-lg">
      <span className="px-2 text-[10px] font-medium uppercase tracking-wide text-gray-9">
        Debug
      </span>
      {STATES.map((s) => (
        <button
          key={s}
          type="button"
          onClick={() => onSelect(s)}
          className={cn(
            "rounded-full px-2 py-1 text-[11px] text-gray-11 transition-colors hover:bg-grayA-3 hover:text-gray-12",
            s === state && "bg-accent-12 text-gray-1 hover:bg-accent-12 hover:text-gray-1",
          )}
        >
          {s}
        </button>
      ))}
    </div>
  );
}

"use client";

import { cn } from "@/lib/utils";
import { useState } from "react";
import {
  DEBUG_DOMAINS,
  DEBUG_STATUSES,
  DEBUG_TRAFFIC,
  DEBUG_VIEWS,
  DEBUG_WINDOWS,
  useOverviewDebug,
} from "./use-overview-debug";

function Segment<T extends string>({
  label,
  value,
  options,
  onChange,
}: {
  label: string;
  value: T;
  options: readonly T[];
  onChange: (v: T) => void;
}) {
  return (
    <div className="flex flex-col gap-1">
      <span className="text-[10px] uppercase tracking-wide font-medium text-gray-9">{label}</span>
      <div className="flex flex-wrap gap-1">
        {options.map((opt) => (
          <button
            key={opt}
            type="button"
            onClick={() => onChange(opt)}
            className={cn(
              "rounded-md px-2 py-1 text-[11px] capitalize border transition-colors",
              value === opt
                ? "bg-accent-12 text-gray-1 border-accent-12"
                : "bg-gray-1 text-accent-12 border-gray-4 hover:bg-gray-3",
            )}
          >
            {opt}
          </button>
        ))}
      </div>
    </div>
  );
}

export function OverviewDebugNav() {
  const [open, setOpen] = useState(false);
  const d = useOverviewDebug();

  if (!open) {
    return (
      <button
        type="button"
        onClick={() => setOpen(true)}
        className="fixed bottom-4 right-4 z-50 rounded-full bg-accent-12 text-gray-1 px-3 py-2 text-xs font-medium shadow-lg hover:opacity-90"
      >
        Debug
      </button>
    );
  }

  return (
    <div className="fixed bottom-4 right-4 z-50 w-64 rounded-xl border border-gray-4 bg-gray-1 p-3 shadow-xl">
      <div className="mb-2 flex items-center justify-between">
        <span className="text-xs font-semibold text-accent-12">Overview · debug</span>
        <button
          type="button"
          onClick={() => setOpen(false)}
          className="text-gray-9 hover:text-accent-12 text-xs"
        >
          ✕
        </button>
      </div>
      <div className="flex flex-col gap-3">
        <Segment label="View" value={d.view} options={DEBUG_VIEWS} onChange={d.setView} />
        <Segment label="Status" value={d.status} options={DEBUG_STATUSES} onChange={d.setStatus} />
        <Segment label="Domain" value={d.domain} options={DEBUG_DOMAINS} onChange={d.setDomain} />
        <Segment
          label="Traffic"
          value={d.traffic}
          options={DEBUG_TRAFFIC}
          onChange={d.setTraffic}
        />
        <Segment label="Window" value={d.win} options={DEBUG_WINDOWS} onChange={d.setWin} />
        <div className="flex flex-col gap-1">
          <span className="text-[10px] uppercase tracking-wide font-medium text-gray-9">
            Rolled back
          </span>
          <div className="flex gap-1">
            {[true, false].map((v) => (
              <button
                key={String(v)}
                type="button"
                onClick={() => d.setRolledBack(v)}
                className={cn(
                  "rounded-md px-2 py-1 text-[11px] border transition-colors",
                  d.rolledBack === v
                    ? "bg-accent-12 text-gray-1 border-accent-12"
                    : "bg-gray-1 text-accent-12 border-gray-4 hover:bg-gray-3",
                )}
              >
                {v ? "on" : "off"}
              </button>
            ))}
          </div>
        </div>
        <button
          type="button"
          onClick={() => {
            d.setView("card");
            d.setStatus("live");
            d.setDomain("custom");
            d.setTraffic("normal");
            d.setWin("auto");
            d.setRolledBack(false);
          }}
          className="mt-1 rounded-md border border-gray-4 px-2 py-1 text-[11px] text-gray-11 hover:bg-gray-3"
        >
          Reset
        </button>
      </div>
    </div>
  );
}

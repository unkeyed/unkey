"use client";

import type { Environment } from "@/lib/collections/deploy/environments";
import { APP_METRICS_WINDOWS, type AppMetricsWindow } from "@unkey/clickhouse";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import { WINDOW_LABELS } from "../lib/series";

export function EnvironmentSelect({
  environments,
  value,
  onChange,
}: {
  environments: Environment[];
  value: string;
  onChange: (id: string) => void;
}) {
  return (
    <Select value={value} onValueChange={(v) => v && onChange(v)}>
      <SelectTrigger wrapperClassName="w-fit" className="h-8 min-w-[160px] text-[13px]">
        <SelectValue>
          {(v: string) => (
            <span className="flex items-center gap-2">
              <span className="size-2 rounded-full bg-success-9" />
              <span className="capitalize">
                {environments.find((e) => e.id === v)?.slug ?? "Environment"}
              </span>
            </span>
          )}
        </SelectValue>
      </SelectTrigger>
      <SelectContent>
        {environments.map((e) => (
          <SelectItem key={e.id} value={e.id} className="capitalize text-[13px]">
            {e.slug}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}

// Quick range pills.
export function WindowPills({
  value,
  onChange,
}: {
  value: AppMetricsWindow;
  onChange: (w: AppMetricsWindow) => void;
}) {
  return (
    <div
      role="radiogroup"
      aria-label="Time range"
      className="inline-flex h-8 items-center rounded-md border border-grayA-4 bg-grayA-2 p-0.5"
    >
      {APP_METRICS_WINDOWS.map((w) => {
        const active = w === value;
        return (
          <button
            key={w}
            type="button"
            role="radio"
            aria-checked={active}
            onClick={() => onChange(w)}
            className={cn(
              "h-7 px-2.5 rounded-[5px] text-[12px] font-medium tabular-nums transition-colors",
              active
                ? "bg-gray-1 text-gray-12 shadow-sm border border-grayA-4"
                : "text-gray-11 hover:text-gray-12",
            )}
          >
            {WINDOW_LABELS[w].short}
          </button>
        );
      })}
    </div>
  );
}

// Range dropdown.
export function WindowSelect({
  value,
  onChange,
}: {
  value: AppMetricsWindow;
  onChange: (w: AppMetricsWindow) => void;
}) {
  return (
    <Select value={value} onValueChange={(v) => v && onChange(v as AppMetricsWindow)}>
      <SelectTrigger wrapperClassName="w-fit" className="h-8 min-w-[150px] text-[13px]">
        <SelectValue>{(v: AppMetricsWindow) => WINDOW_LABELS[v]?.long ?? v}</SelectValue>
      </SelectTrigger>
      <SelectContent>
        {APP_METRICS_WINDOWS.map((w) => (
          <SelectItem key={w} value={w} className="text-[13px]">
            {WINDOW_LABELS[w].long}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}

export function Legend({
  items,
}: {
  items: { label: string; color: string; value?: string }[];
}) {
  return (
    <div className="flex items-center gap-3 flex-wrap">
      {items.map((it) => (
        <div key={it.label} className="flex items-center gap-1.5 text-[11px] text-gray-11">
          <span className="size-2 rounded-[2px]" style={{ backgroundColor: it.color }} />
          <span>{it.label}</span>
          {it.value && <span className="font-mono tabular-nums text-gray-12">{it.value}</span>}
        </div>
      ))}
    </div>
  );
}

"use client";

import { cn } from "@/lib/utils";
import { IconChevronRightOutline12 } from "@unkey/icons";
import { useCallback, useEffect, useState } from "react";
import { type VariantId, VARIANTS } from "./registry";
import type { Density, LaunchpadOptions, SparkMode } from "./types";

const STORAGE_KEY = "unkey.launchpad.options";

export type PickerState = LaunchpadOptions & { variant: VariantId };

const DEFAULTS: PickerState = {
  variant: "flat",
  spark: "bars",
  density: "default",
  showProject: true,
  forceEmpty: false,
};

export function useLaunchpadPicker() {
  const [state, setState] = useState<PickerState>(DEFAULTS);

  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    const fromUrl = params.get("lp");
    const stored = window.localStorage.getItem(STORAGE_KEY);
    const base = stored ? { ...DEFAULTS, ...(JSON.parse(stored) as Partial<PickerState>) } : DEFAULTS;
    setState(fromUrl ? { ...base, variant: fromUrl as VariantId } : base);
  }, []);

  const update = useCallback((patch: Partial<PickerState>) => {
    setState((current) => {
      const next = { ...current, ...patch };
      window.localStorage.setItem(STORAGE_KEY, JSON.stringify(next));
      const url = new URL(window.location.href);
      url.searchParams.set("lp", next.variant);
      window.history.replaceState(null, "", url);
      return next;
    });
  }, []);

  return { state, update };
}

function Group({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="flex items-center justify-between gap-3 px-3 py-1.5">
      <span className="text-[11px] uppercase tracking-wide text-gray-9">{label}</span>
      <div className="flex gap-1">{children}</div>
    </div>
  );
}

function Chip({
  active,
  onClick,
  children,
}: {
  active: boolean;
  onClick: () => void;
  children: React.ReactNode;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={cn(
        "rounded px-1.5 py-0.5 text-[11px] transition-colors",
        active ? "bg-grayA-4 text-accent-12" : "text-gray-9 hover:bg-grayA-2",
      )}
    >
      {children}
    </button>
  );
}

export function LaunchpadPicker({
  state,
  update,
}: {
  state: PickerState;
  update: (patch: Partial<PickerState>) => void;
}) {
  const [open, setOpen] = useState(false);
  const active = VARIANTS.find((variant) => variant.id === state.variant) ?? VARIANTS[0];

  return (
    <div className="fixed bottom-4 right-4 z-50 w-[300px] text-[13px]">
      {open && (
        <div className="mb-2 max-h-[70vh] overflow-y-auto rounded-lg border border-grayA-6 bg-background shadow-lg">
          <div className="border-b border-grayA-4 px-3 py-2 text-[11px] uppercase tracking-wide text-gray-9">
            Variant
          </div>
          {VARIANTS.map((variant) => (
            <button
              key={variant.id}
              type="button"
              onClick={() => update({ variant: variant.id })}
              className={cn(
                "flex w-full flex-col gap-0.5 border-b border-grayA-3 px-3 py-2 text-left transition-colors hover:bg-grayA-2",
                variant.id === state.variant && "bg-grayA-3",
              )}
            >
              <span className="text-accent-12">{variant.name}</span>
              <span className="text-xs leading-snug text-gray-9">{variant.note}</span>
            </button>
          ))}
          <div className="py-1">
            <Group label="Sparkline">
              {(["off", "bars", "hover"] as SparkMode[]).map((mode) => (
                <Chip key={mode} active={state.spark === mode} onClick={() => update({ spark: mode })}>
                  {mode}
                </Chip>
              ))}
            </Group>
            <Group label="Density">
              {(["compact", "default", "roomy"] as Density[]).map((density) => (
                <Chip
                  key={density}
                  active={state.density === density}
                  onClick={() => update({ density })}
                >
                  {density === "compact" ? "28" : density === "default" ? "32" : "40"}
                </Chip>
              ))}
            </Group>
            <Group label="Project chip">
              <Chip active={state.showProject} onClick={() => update({ showProject: true })}>
                on
              </Chip>
              <Chip active={!state.showProject} onClick={() => update({ showProject: false })}>
                off
              </Chip>
            </Group>
            <Group label="Face">
              <Chip active={!state.forceEmpty} onClick={() => update({ forceEmpty: false })}>
                migrated
              </Chip>
              <Chip active={state.forceEmpty} onClick={() => update({ forceEmpty: true })}>
                new
              </Chip>
            </Group>
          </div>
        </div>
      )}
      <button
        type="button"
        onClick={() => setOpen(!open)}
        className="flex h-8 w-full items-center gap-2 rounded-lg border border-grayA-6 bg-background px-3 shadow-sm transition-colors hover:bg-grayA-2"
      >
        <span className="min-w-0 flex-1 truncate text-left text-accent-12">{active.name}</span>
        <span className="shrink-0 text-xs text-gray-9">{`${VARIANTS.indexOf(active) + 1}/${VARIANTS.length}`}</span>
        <IconChevronRightOutline12
          className={cn("size-3 shrink-0 text-gray-9 transition-transform", open && "rotate-90")}
        />
      </button>
    </div>
  );
}

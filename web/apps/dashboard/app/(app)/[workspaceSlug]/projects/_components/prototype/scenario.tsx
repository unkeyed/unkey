"use client";

import { cn } from "@/lib/utils";
import { useSearchParams } from "next/navigation";
import { useCallback, useEffect, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { SCENARIO_LABELS, type Scenario } from "./mock-data";

const SCENARIOS: Scenario[] = ["new", "migrated", "active"];

export type RowVariant = "detailed" | "graph" | "flat" | "list" | "tile" | "hybrid" | "metric";

// All prototype choices live in the URL query string so a configuration can be
// shared as a link (e.g. ?scenario=migrated),
// and are mirrored to localStorage so they survive navigating away and back.
// URL param wins; stored value fills in when the URL has none.

// Prefix matches SCENARIO_STORAGE_KEY in store.tsx: "scenario" lands on the
// exact key the tRPC interceptor reads.
const STORAGE_PREFIX = "unkey.projects-prototype.";

function readStored(key: string): string | null {
  try {
    return localStorage.getItem(STORAGE_PREFIX + key);
  } catch {
    return null;
  }
}

function writeStored(key: string, value: string) {
  try {
    localStorage.setItem(STORAGE_PREFIX + key, value);
  } catch {
    // ignore
  }
}

const CONFIG_EVENT = "prototype:config";

function writeParam(key: string, value: string) {
  try {
    const url = new URL(window.location.href);
    url.searchParams.set(key, value);
    window.history.replaceState(null, "", url.toString());
    window.dispatchEvent(new Event(CONFIG_EVENT));
  } catch {
    // ignore
  }
}

// Current query string (including the prototype config params), so in-prototype
// links can carry the chosen configuration across pages.
export function useCurrentSearch(): string {
  const [search, setSearch] = useState("");
  useEffect(() => {
    const read = () => setSearch(window.location.search);
    read();
    window.addEventListener(CONFIG_EVENT, read);
    window.addEventListener("popstate", read);
    return () => {
      window.removeEventListener(CONFIG_EVENT, read);
      window.removeEventListener("popstate", read);
    };
  }, []);
  return search;
}

// URL wins whenever it carries a valid value, and stays authoritative across
// CLIENT-SIDE navigations too (useSearchParams is reactive) — a mount-only read
// left the page showing the previous config under a freshly pasted URL after a
// soft navigation. localStorage fills in when the URL has no value and carries
// the choice to pages the user reaches without params.
function useUrlEnum<T extends string>(key: string, allowed: readonly T[], initial: T) {
  const searchParams = useSearchParams();
  const raw = searchParams?.get(key) ?? null;
  const urlValue = raw && (allowed as readonly string[]).includes(raw) ? (raw as T) : null;

  // Only consulted while the URL has no valid value for this key.
  const [fallback, setFallback] = useState<T>(initial);

  useEffect(() => {
    if (urlValue) {
      writeStored(key, urlValue);
      return;
    }
    const stored = readStored(key);
    const resolved =
      stored && (allowed as readonly string[]).includes(stored) ? (stored as T) : initial;
    setFallback(resolved);
    writeParam(key, resolved);
    writeStored(key, resolved);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [key, urlValue]);

  const set = useCallback(
    (next: T) => {
      setFallback(next);
      writeParam(key, next);
      writeStored(key, next);
    },
    [key],
  );

  return [urlValue ?? fallback, set] as const;
}

export function useScenario() {
  const [scenario, setScenario] = useUrlEnum<Scenario>("scenario", SCENARIOS, "migrated");
  return { scenario, setScenario };
}

export type OverviewOption = "hero" | "stats" | "hub" | "hybrid";
export const OVERVIEW_OPTIONS: OverviewOption[] = ["hero", "stats", "hub", "hybrid"];
export const OVERVIEW_OPTION_LABELS: Record<OverviewOption, string> = {
  hero: "Deploy hero",
  stats: "Stat tiles",
  hub: "Apps + keys hub",
  hybrid: "Hybrid rows",
};

export function useOverviewOption() {
  const [option, setOption] = useUrlEnum<OverviewOption>(
    "overviewOption",
    OVERVIEW_OPTIONS,
    "hero",
  );
  return { option, setOption };
}

// Two competing treatments for the hybrid option's metric rows: Andreas's
// original full-bleed background chart vs the contained chart we settled on
// for the rail. Kept as a toggle so both can be judged side by side.
export type HybridStyle = "bleed" | "contained";
export const HYBRID_STYLES: HybridStyle[] = ["bleed", "contained"];
export const HYBRID_STYLE_LABELS: Record<HybridStyle, string> = {
  bleed: "Full-bleed chart",
  contained: "Contained chart",
};

export function useHybridStyle() {
  const [hybridStyle, setHybridStyle] = useUrlEnum<HybridStyle>("hybrid", HYBRID_STYLES, "bleed");
  return { hybridStyle, setHybridStyle };
}

export type Cmd = { id: string; group: string; label: string; active: boolean; run: () => void };
export type CmdGroup = { name: string; items: Cmd[] };

// Shared shell behind every "Prototype options" pill: a bottom-right trigger
// plus a filterable command list, grouped by the caller. DebugCommand (below)
// and the overview page's own switcher both just build `groups` and render this.
export function PrototypeCommandPalette({ groups }: { groups: CmdGroup[] }) {
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState("");
  const [activeIndex, setActiveIndex] = useState(0);
  const inputRef = useRef<HTMLInputElement>(null);

  // No ⌘K binding — that collides with the app's own command menu. Open via the
  // pill; Escape closes. Swallow ⌘K while open so it doesn't also pop the real
  // command menu on top of this one.
  useEffect(() => {
    if (!open) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        setOpen(false);
      } else if (e.key === "k" && (e.metaKey || e.ctrlKey)) {
        e.preventDefault();
        e.stopPropagation();
      }
    };
    window.addEventListener("keydown", onKey, true);
    return () => window.removeEventListener("keydown", onKey, true);
  }, [open]);

  useEffect(() => {
    if (open) {
      setQuery("");
      setActiveIndex(0);
      requestAnimationFrame(() => inputRef.current?.focus());
    }
  }, [open]);

  const cmds = groups.flatMap((g) => g.items);
  const q = query.trim().toLowerCase();
  const filtered = q ? cmds.filter((c) => `${c.group} ${c.label}`.toLowerCase().includes(q)) : cmds;

  const run = (c: Cmd | undefined) => {
    if (!c) return;
    c.run();
    setOpen(false);
  };

  const filteredGroups: CmdGroup[] = [];
  for (const c of filtered) {
    let g = filteredGroups.find((x) => x.name === c.group);
    if (!g) {
      g = { name: c.group, items: [] };
      filteredGroups.push(g);
    }
    g.items.push(c);
  }

  return (
    <>
      <button
        type="button"
        onClick={() => setOpen(true)}
        className="fixed bottom-4 right-4 z-40 inline-flex items-center gap-1.5 rounded-full border border-grayA-6 bg-gray-1 px-3 py-1.5 text-[11px] font-medium text-gray-11 shadow-lg transition-colors hover:text-gray-12"
      >
        Prototype options
      </button>
      {open &&
        createPortal(
          // biome-ignore lint/a11y/useKeyWithClickEvents: prototype backdrop
          <div
            className="fixed inset-0 z-50 flex items-start justify-center bg-black/30 backdrop-blur-xs pt-[15vh]"
            onClick={() => setOpen(false)}
          >
            {/* biome-ignore lint/a11y/noStaticElementInteractions: stops backdrop close */}
            <div
              className="w-[440px] max-w-[90vw] overflow-hidden rounded-xl border border-grayA-4 bg-background shadow-2xl"
              onClick={(e) => e.stopPropagation()}
            >
              <input
                ref={inputRef}
                value={query}
                onChange={(e) => {
                  setQuery(e.target.value);
                  setActiveIndex(0);
                }}
                onKeyDown={(e) => {
                  if (e.key === "ArrowDown") {
                    e.preventDefault();
                    setActiveIndex((a) => Math.min(a + 1, filtered.length - 1));
                  } else if (e.key === "ArrowUp") {
                    e.preventDefault();
                    setActiveIndex((a) => Math.max(a - 1, 0));
                  } else if (e.key === "Enter") {
                    e.preventDefault();
                    run(filtered[activeIndex]);
                  }
                }}
                placeholder="Set scenario…"
                className="w-full border-b border-grayA-4 bg-transparent px-4 py-3 text-sm text-accent-12 outline-none placeholder:text-gray-9"
              />
              <div className="max-h-[50vh] overflow-y-auto p-1.5">
                {filtered.length === 0 ? (
                  <div className="px-3 py-6 text-center text-xs text-gray-9">No matches</div>
                ) : (
                  filteredGroups.map((g) => (
                    <div key={g.name} className="mb-1">
                      <div className="px-2 pt-2 pb-1 text-[10px] font-medium uppercase tracking-wide text-gray-9">
                        {g.name}
                      </div>
                      {g.items.map((c) => {
                        const idx = filtered.indexOf(c);
                        return (
                          <button
                            key={c.id}
                            type="button"
                            onMouseMove={() => setActiveIndex(idx)}
                            onClick={() => run(c)}
                            className={cn(
                              "flex w-full items-center justify-between gap-2 rounded-md px-2 py-1.5 text-left text-[13px] text-accent-12",
                              idx === activeIndex ? "bg-grayA-3" : "hover:bg-grayA-2",
                            )}
                          >
                            <span>{c.label}</span>
                            {c.active && <span className="text-[11px] text-gray-9">current</span>}
                          </button>
                        );
                      })}
                    </div>
                  ))
                )}
              </div>
            </div>
          </div>,
          document.body,
        )}
    </>
  );
}

// The projects list keeps one choice — which scenario's data to show — plus a
// reset. Mark, row style, agent style and chart colours are settled and live as
// constants at their call sites.
export function DebugCommand({
  scenario,
  onScenario,
  onReset,
}: {
  scenario: Scenario;
  onScenario: (s: Scenario) => void;
  onReset: () => void;
}) {
  const groups: CmdGroup[] = [
    {
      name: "Scenario",
      items: SCENARIOS.map((s) => ({
        id: `sc-${s}`,
        group: "Scenario",
        label: SCENARIO_LABELS[s],
        active: s === scenario,
        run: () => onScenario(s),
      })),
    },
    {
      name: "Data",
      items: [
        { id: "reset", group: "Data", label: "Reset prototype data", active: false, run: onReset },
      ],
    },
  ];

  return <PrototypeCommandPalette groups={groups} />;
}

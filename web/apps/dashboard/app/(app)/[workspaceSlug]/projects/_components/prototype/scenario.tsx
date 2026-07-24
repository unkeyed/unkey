"use client";

import { cn } from "@/lib/utils";
import { useCallback, useEffect, useRef, useState } from "react";
import { createPortal } from "react-dom";
import type { AgentStyle } from "./agent-setup";
import type { Mark } from "./marks";
import { SCENARIO_LABELS, type Scenario } from "./mock-data";

const SCENARIOS: Scenario[] = ["new", "migrated", "active"];

export type RowVariant = "detailed" | "graph" | "flat" | "list" | "tile" | "hybrid";

const ROW_VARIANTS: RowVariant[] = ["detailed", "graph", "flat", "list", "tile", "hybrid"];
const ROW_VARIANT_LABELS: Record<RowVariant, string> = {
  detailed: "Detailed",
  graph: "Graph",
  flat: "Flat",
  list: "List",
  tile: "Tile",
  hybrid: "Hybrid",
};

const MARKS: Mark[] = ["line", "bars", "ratio", "heatmap"];
const MARK_LABELS: Record<Mark, string> = {
  line: "Line",
  bars: "Bars",
  ratio: "Ratio",
  heatmap: "Heatmap",
};

const AGENT_STYLES: AgentStyle[] = ["minimal", "stacked"];
const AGENT_STYLE_LABELS: Record<AgentStyle, string> = {
  minimal: "Minimal",
  stacked: "Stacked",
};

// All prototype choices live in the URL query string so a configuration can be
// shared as a link (e.g. ?scenario=migrated&row=list&mark=bars&agent=minimal),
// and are mirrored to localStorage so they survive navigating away and back.
// URL param wins; stored value fills in when the URL has none.
function readParam(key: string): string | null {
  try {
    return new URLSearchParams(window.location.search).get(key);
  } catch {
    return null;
  }
}

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

// Reads the param on mount (URL wins, then localStorage, then `initial`) and
// writes the resolved value back to both, so the URL encodes the full config
// even for defaults and the choice survives navigation.
function useUrlEnum<T extends string>(key: string, allowed: readonly T[], initial: T) {
  const [value, setValue] = useState<T>(initial);

  useEffect(() => {
    const valid = (v: string | null) =>
      v && (allowed as readonly string[]).includes(v) ? (v as T) : null;
    const resolved = valid(readParam(key)) ?? valid(readStored(key)) ?? initial;
    if (resolved !== initial) {
      setValue(resolved);
    }
    writeParam(key, resolved);
    writeStored(key, resolved);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const set = useCallback(
    (next: T) => {
      setValue(next);
      writeParam(key, next);
      writeStored(key, next);
    },
    [key],
  );

  return [value, set] as const;
}

function useUrlBool(key: string, initial: boolean) {
  const [value, setValue] = useState(initial);

  useEffect(() => {
    const param = readParam(key) ?? readStored(key);
    const resolved = param == null ? initial : param === "true";
    if (resolved !== initial) {
      setValue(resolved);
    }
    writeParam(key, String(resolved));
    writeStored(key, String(resolved));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const set = useCallback(
    (next: boolean) => {
      setValue(next);
      writeParam(key, String(next));
      writeStored(key, String(next));
    },
    [key],
  );

  return [value, set] as const;
}

export function useScenario() {
  const [scenario, setScenario] = useUrlEnum<Scenario>("scenario", SCENARIOS, "migrated");
  return { scenario, setScenario };
}

export function useRowVariant() {
  const [variant, setVariant] = useUrlEnum<RowVariant>("row", ROW_VARIANTS, "list");
  return { variant, setVariant };
}

export function useMark() {
  const [mark, setMark] = useUrlEnum<Mark>("mark", MARKS, "bars");
  return { mark, setMark };
}

export function useAgentStyle() {
  const [agentStyle, setAgentStyle] = useUrlEnum<AgentStyle>("agent", AGENT_STYLES, "minimal");
  return { agentStyle, setAgentStyle };
}

export function useAgentDismissed() {
  return useUrlBool("agentHidden", false);
}

type Cmd = { id: string; group: string; label: string; active: boolean; run: () => void };

// Single ⌘K palette that replaces the separate scenario/row-style/mark toggle
// bars — prototype-only debug control.
export function DebugCommand({
  scenario,
  onScenario,
  variant,
  onVariant,
  mark,
  onMark,
  agentStyle,
  onAgentStyle,
  agentDismissed,
  onToggleAgent,
  onReset,
}: {
  scenario: Scenario;
  onScenario: (s: Scenario) => void;
  variant: RowVariant;
  onVariant: (v: RowVariant) => void;
  mark: Mark;
  onMark: (m: Mark) => void;
  agentStyle: AgentStyle;
  onAgentStyle: (a: AgentStyle) => void;
  agentDismissed: boolean;
  onToggleAgent: () => void;
  onReset: () => void;
}) {
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

  const cmds: Cmd[] = [
    ...SCENARIOS.map((s) => ({
      id: `sc-${s}`,
      group: "Scenario",
      label: SCENARIO_LABELS[s],
      active: s === scenario,
      run: () => onScenario(s),
    })),
    ...ROW_VARIANTS.map((v) => ({
      id: `rv-${v}`,
      group: "Row style",
      label: ROW_VARIANT_LABELS[v],
      active: v === variant,
      run: () => onVariant(v),
    })),
    ...MARKS.map((m) => ({
      id: `mk-${m}`,
      group: "Mark",
      label: MARK_LABELS[m],
      active: m === mark,
      run: () => onMark(m),
    })),
    ...AGENT_STYLES.map((a) => ({
      id: `ag-${a}`,
      group: "Agent style",
      label: AGENT_STYLE_LABELS[a],
      active: a === agentStyle,
      run: () => onAgentStyle(a),
    })),
    {
      id: "agent",
      group: "Agent card",
      label: agentDismissed ? "Restore agent card" : "Dismiss agent card",
      active: false,
      run: onToggleAgent,
    },
    {
      id: "reset",
      group: "Data",
      label: "Reset prototype data",
      active: false,
      run: onReset,
    },
  ];

  const q = query.trim().toLowerCase();
  const filtered = q ? cmds.filter((c) => `${c.group} ${c.label}`.toLowerCase().includes(q)) : cmds;

  const run = (c: Cmd | undefined) => {
    if (!c) return;
    c.run();
    setOpen(false);
  };

  const groups: { name: string; items: Cmd[] }[] = [];
  for (const c of filtered) {
    let g = groups.find((x) => x.name === c.group);
    if (!g) {
      g = { name: c.group, items: [] };
      groups.push(g);
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
                placeholder="Set scenario, row style, mark…"
                className="w-full border-b border-grayA-4 bg-transparent px-4 py-3 text-sm text-accent-12 outline-none placeholder:text-gray-9"
              />
              <div className="max-h-[50vh] overflow-y-auto p-1.5">
                {filtered.length === 0 ? (
                  <div className="px-3 py-6 text-center text-xs text-gray-9">No matches</div>
                ) : (
                  groups.map((g) => (
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

"use client";

import {
  createContext,
  type ReactNode,
  useCallback,
  useContext,
  useEffect,
  useState,
} from "react";
import { type KeyspaceStat, MOCK, type OverviewData, type Scenario } from "./mock-data";

// The whole prototype dataset lives client-side in localStorage so the preview
// deployment needs no database and every signed-in user sees the same world.
// The seed is deterministic (fixed PRNG), so first paint matches across users
// and across SSR/client hydration.

export type MockKey = {
  id: string;
  name: string;
  // Visible key prefix, e.g. "prod_3f9a" — mirrors the real `start` column.
  start: string;
  requests24h: number;
  validPct: number;
  lastUsedMinAgo: number;
  enabled: boolean;
  spark: number[];
};

export type World = OverviewData & {
  // Keys per keyspace id, for the project-scoped keyspace detail page.
  keys: Record<string, MockKey[]>;
};

export type Worlds = Record<Scenario, World>;

// Mirrors the ?scenario= URL param so non-React code (the prototype tRPC
// interceptor) can serve the same scenario's data on other routes.
export const SCENARIO_STORAGE_KEY = "unkey.projects-prototype.scenario";

const STORAGE_KEY = "unkey.projects-prototype";
// Bump when the seed shape changes so stale localStorage copies get replaced.
const VERSION = 2;

export function mulberry32(seed: number) {
  let a = seed;
  return () => {
    a |= 0;
    a = (a + 0x6d2b79f5) | 0;
    let t = Math.imul(a ^ (a >>> 15), 1 | a);
    t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t;
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
  };
}

export function hashCode(s: string): number {
  let h = 0;
  for (let i = 0; i < s.length; i++) {
    h = (Math.imul(h, 31) + s.charCodeAt(i)) | 0;
  }
  return h;
}

const KEY_NAMES = [
  "default",
  "ci-runner",
  "vercel-preview",
  "staging-smoke",
  "customer-globex",
  "customer-initech",
  "customer-umbrella",
  "partner-zapier",
  "internal-cron",
  "webhook-relay",
  "mobile-app",
  "docs-playground",
  "load-test",
  "cli-local",
];

function generateKeys(ks: KeyspaceStat): MockKey[] {
  const rand = mulberry32(hashCode(ks.id));
  const count = Math.max(5, Math.min(12, Math.round(ks.keyCount / 200) + 5));

  const names = [...KEY_NAMES];
  for (let i = names.length - 1; i > 0; i--) {
    const j = Math.floor(rand() * (i + 1));
    [names[i], names[j]] = [names[j], names[i]];
  }

  const weights = Array.from({ length: count }, () => rand() ** 2 + 0.02);
  const weightSum = weights.reduce((a, b) => a + b, 0);
  const base = ks.spark["24h"];

  const hex = (len: number) =>
    Array.from({ length: len }, () => Math.floor(rand() * 16).toString(16)).join("");

  const keys = weights.map((weight, i) => {
    const share = weight / weightSum;
    const requests24h = Math.round(ks.requests["24h"] * share);
    const validPct = Math.min(100, Math.round((ks.validPct + (rand() * 2 - 1) * 1.5) * 10) / 10);
    return {
      id: `key_${hex(12)}`,
      name: names[i] ?? `service-${i}`,
      start: `${ks.name.replace(/[^a-z0-9]/gi, "").slice(0, 4) || "key"}_${hex(4)}`,
      requests24h,
      validPct,
      lastUsedMinAgo:
        requests24h > 1_000 ? 1 + Math.floor(rand() * 15) : Math.floor(rand() * 60 * 36),
      enabled: rand() > 0.08,
      spark: base.map((v) => Math.max(0, Math.round(v * share * 100 * (0.7 + rand() * 0.6)))),
    };
  });

  return keys.sort((a, b) => b.requests24h - a.requests24h);
}

export function seedWorlds(): Worlds {
  const worlds = {} as Worlds;
  for (const scenario of Object.keys(MOCK) as Scenario[]) {
    const data = MOCK[scenario];
    const keys: Record<string, MockKey[]> = {};
    for (const ks of data.keyspaces) {
      keys[ks.id] = generateKeys(ks);
    }
    worlds[scenario] = { ...data, keys };
  }
  return worlds;
}

type PrototypeContext = {
  worlds: Worlds;
  updateWorld: (scenario: Scenario, update: (world: World) => World) => void;
  resetWorlds: () => void;
};

const Context = createContext<PrototypeContext | null>(null);

function save(worlds: Worlds) {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify({ v: VERSION, worlds }));
  } catch {
    // storage unavailable — stay in-memory
  }
}

// Non-React accessor for code that runs outside the provider (the prototype
// tRPC interceptor). Reads persisted edits when present, otherwise the seed.
export function loadWorlds(): Worlds {
  if (typeof window !== "undefined") {
    try {
      const raw = localStorage.getItem(STORAGE_KEY);
      if (raw) {
        const parsed = JSON.parse(raw) as { v?: number; worlds?: Worlds };
        if (parsed.v === VERSION && parsed.worlds) {
          return parsed.worlds;
        }
      }
    } catch {
      // fall through to the seed
    }
  }
  return seedWorlds();
}

export function PrototypeProvider({ children }: { children: ReactNode }) {
  const [worlds, setWorlds] = useState<Worlds>(seedWorlds);

  // localStorage is only readable after mount; the deterministic seed keeps the
  // server and first client render identical, then any persisted edits win.
  useEffect(() => {
    try {
      const raw = localStorage.getItem(STORAGE_KEY);
      if (raw) {
        const parsed = JSON.parse(raw) as { v?: number; worlds?: Worlds };
        if (parsed.v === VERSION && parsed.worlds) {
          setWorlds(parsed.worlds);
          return;
        }
      }
    } catch {
      // fall through to reseeding
    }
    save(seedWorlds());
  }, []);

  const updateWorld = useCallback((scenario: Scenario, update: (world: World) => World) => {
    setWorlds((prev) => {
      const next = { ...prev, [scenario]: update(prev[scenario]) };
      save(next);
      return next;
    });
  }, []);

  const resetWorlds = useCallback(() => {
    try {
      localStorage.removeItem(STORAGE_KEY);
    } catch {
      // ignore
    }
    const fresh = seedWorlds();
    save(fresh);
    setWorlds(fresh);
  }, []);

  return <Context.Provider value={{ worlds, updateWorld, resetWorlds }}>{children}</Context.Provider>;
}

export function usePrototypeWorlds(): PrototypeContext {
  const ctx = useContext(Context);
  if (!ctx) {
    throw new Error("usePrototypeWorlds must be used inside PrototypeProvider");
  }
  return ctx;
}

// Locates a keyspace by id across all scenario worlds, so detail pages don't
// need the scenario threaded through the URL.
export function findKeyspace(worlds: Worlds, keyspaceId: string) {
  for (const scenario of Object.keys(worlds) as Scenario[]) {
    const world = worlds[scenario];
    const keyspace = world.keyspaces.find((ks) => ks.id === keyspaceId);
    if (keyspace) {
      return { scenario, world, keyspace, keys: world.keys[keyspaceId] ?? [] };
    }
  }
  return null;
}

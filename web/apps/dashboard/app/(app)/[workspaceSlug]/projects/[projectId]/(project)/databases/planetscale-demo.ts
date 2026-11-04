"use client";

import { useSyncExternalStore } from "react";
import type { MetricPoint, PlanetscaleDatabase, PlanetscaleOrg } from "./database-model";

const INITIALIZE_MS = 8_000;
const POLL_MS = 1_000;
const HOUR_MS = 3_600_000;
const DAY_MS = 24 * HOUR_MS;
const ACTIVITY_POINTS = 48;

function activity(now: number, base: number, seed: number): MetricPoint[] {
  const stepMs = DAY_MS / ACTIVITY_POINTS;
  return Array.from({ length: ACTIVITY_POINTS }, (_, i) => {
    const day = Math.sin((i / ACTIVITY_POINTS) * Math.PI * 2 - Math.PI / 2 + seed) * 0.35;
    const noise = Math.sin(i * 12.9898 * seed) * 0.12;
    const at = now - (ACTIVITY_POINTS - 1 - i) * stepMs;
    return [Math.floor(at / 1000), Math.max(0, Math.round(base * (1 + day + noise) * 10) / 10)];
  });
}

export const PLANETSCALE_ORGS: PlanetscaleOrg[] = [
  { name: "acme", hasCard: true },
  { name: "acme-staging", hasCard: false },
];

type DemoPlanetscale = {
  connected: boolean;
  databases: PlanetscaleDatabase[];
};

function seed(): DemoPlanetscale {
  const now = Date.now();
  return {
    connected: false,
    databases: [
      {
        name: "main",
        org: "acme",
        engine: "postgres",
        region: "us-east-1",
        cluster: "ha",
        size: "PS-20",
        state: "ready",
        defaultBranch: "main",
        branches: ["main", "dev"],
        createdAt: now - 42 * DAY_MS,
        tableCount: 38,
        schemaUpdatedAt: now - 3 * HOUR_MS,
        lastBackupAt: now - 2 * HOUR_MS,
        maintenanceAt: now + 4 * DAY_MS,
        vitals: {
          queries: activity(now, 142, 1),
          latencyP95Ms: 4.2,
          cpuRatio: 0.2,
          connections: 32,
        },
      },
      {
        name: "analytics",
        org: "acme",
        engine: "mysql",
        region: "eu-central-1",
        cluster: "ha",
        size: "PS-10",
        state: "ready",
        defaultBranch: "main",
        branches: ["main"],
        createdAt: now - 9 * DAY_MS,
        tableCount: 12,
        schemaUpdatedAt: now - DAY_MS,
        lastBackupAt: now - 5 * HOUR_MS,
        maintenanceAt: now + 11 * DAY_MS,
        vitals: {
          queries: activity(now, 36, 2),
          latencyP95Ms: 11.8,
          cpuRatio: 0.07,
          connections: 8,
        },
      },
      {
        name: "billing",
        org: "acme",
        engine: "postgres",
        region: "eu-west-2",
        cluster: "single",
        size: "PS-5",
        state: "ready",
        defaultBranch: "main",
        branches: ["main", "dev"],
        createdAt: now - 3 * DAY_MS,
        tableCount: 9,
        schemaUpdatedAt: now - 6 * DAY_MS,
        lastBackupAt: now - 20 * HOUR_MS,
        maintenanceAt: now + 2 * DAY_MS,
        vitals: {
          queries: activity(now, 4, 3),
          latencyP95Ms: 2.1,
          cpuRatio: 0.03,
          connections: 3,
        },
      },
    ],
  };
}

function settle(state: DemoPlanetscale, now: number): DemoPlanetscale {
  const due = (db: PlanetscaleDatabase) =>
    db.state === "provisioning" && now >= db.createdAt + INITIALIZE_MS;
  if (!state.databases.some(due)) {
    return state;
  }
  return {
    ...state,
    databases: state.databases.map((db) => (due(db) ? { ...db, state: "ready" } : db)),
  };
}

let state: DemoPlanetscale | null = null;
const listeners = new Set<() => void>();

function read(): DemoPlanetscale {
  state ??= seed();
  return state;
}

function write(next: DemoPlanetscale) {
  state = next;
  for (const listener of listeners) {
    listener();
  }
}

function subscribe(listener: () => void) {
  listeners.add(listener);
  const timer = window.setInterval(() => {
    const current = read();
    const next = settle(current, Date.now());
    if (next !== current) {
      write(next);
    }
  }, POLL_MS);
  return () => {
    listeners.delete(listener);
    window.clearInterval(timer);
  };
}

export function usePlanetscaleStore(): DemoPlanetscale | null {
  return useSyncExternalStore(subscribe, read, () => null);
}

export function connectPlanetscale() {
  write({ ...read(), connected: true });
}

export function disconnectPlanetscale() {
  write({ ...read(), connected: false });
}

export function findPlanetscaleDatabase(org: string, name: string): PlanetscaleDatabase {
  const db = read().databases.find((d) => d.org === org && d.name === name);
  if (!db) {
    throw new Error(`${org} has no database named ${name}.`);
  }
  return db;
}

export function createPlanetscaleDatabase(
  input: Pick<
    PlanetscaleDatabase,
    "org" | "name" | "engine" | "region" | "cluster" | "size" | "branches"
  >,
): PlanetscaleDatabase {
  const current = read();
  const db: PlanetscaleDatabase = {
    ...input,
    state: "provisioning",
    defaultBranch: "main",
    branches: [...new Set(["main", ...input.branches])],
    createdAt: Date.now(),
    tableCount: 0,
    schemaUpdatedAt: null,
    lastBackupAt: null,
    maintenanceAt: null,
    vitals: {
      queries: [],
      latencyP95Ms: 0,
      cpuRatio: 0,
      connections: 0,
    },
  };
  write({ ...current, databases: [...current.databases, db] });
  return db;
}

export function addPlanetscaleBranches(
  { org, name }: { org: string; name: string },
  branches: string[],
): PlanetscaleDatabase {
  const current = read();
  const db = findPlanetscaleDatabase(org, name);
  const next = { ...db, branches: [...new Set([...db.branches, ...branches])] };
  write({ ...current, databases: current.databases.map((d) => (d === db ? next : d)) });
  return next;
}

"use client";

import { useSyncExternalStore } from "react";
import type { PlanetscaleDatabase, PlanetscaleOrg } from "./database-model";

const INITIALIZE_MS = 8_000;
const POLL_MS = 1_000;
const DAY_MS = 86_400_000;

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

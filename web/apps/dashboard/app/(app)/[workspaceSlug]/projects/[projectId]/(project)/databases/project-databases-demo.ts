"use client";

import { match } from "@unkey/match";
import { useMemo, useSyncExternalStore } from "react";
import {
  type CreateDatabaseInput,
  type Database,
  type DatabaseLink,
  type EnvironmentBranches,
  type PlanetscaleDatabase,
  environmentBranch,
} from "./database-model";
import {
  addPlanetscaleBranches,
  createPlanetscaleDatabase,
  findPlanetscaleDatabase,
  usePlanetscaleStore,
} from "./planetscale-demo";

type Store = Map<string, DatabaseLink[]>;

let store: Store = new Map();
const listeners = new Set<() => void>();
const EMPTY: DatabaseLink[] = [];

function emit(next: Store) {
  store = next;
  for (const listener of listeners) {
    listener();
  }
}

function subscribe(listener: () => void) {
  listeners.add(listener);
  return () => listeners.delete(listener);
}

function setProject(projectId: string, update: (dbs: DatabaseLink[]) => DatabaseLink[]) {
  const next = new Map(store);
  next.set(projectId, update(store.get(projectId) ?? EMPTY));
  emit(next);
}

function linkOf(
  db: PlanetscaleDatabase,
  branches: EnvironmentBranches,
  link: Pick<DatabaseLink, "id" | "createdAt">,
): DatabaseLink {
  return {
    ...link,
    name: db.name,
    engine: db.engine,
    region: db.region,
    planetscale: { org: db.org, branches },
  };
}

const SEED_LINKS = [
  { org: "acme", name: "main", preview: "dev" },
  { org: "acme", name: "analytics", preview: "main" },
  { org: "acme", name: "billing", preview: "dev" },
];

function seed(): DatabaseLink[] {
  return SEED_LINKS.map(({ org, name, preview }) => {
    const db = findPlanetscaleDatabase(org, name);
    return linkOf(
      db,
      {
        production: environmentBranch(db, db.defaultBranch),
        preview: environmentBranch(db, preview),
      },
      { id: `db_${name}`, createdAt: db.createdAt },
    );
  });
}

export function seedDatabases(projectId: string) {
  if (!store.has(projectId)) {
    setProject(projectId, seed);
  }
}

export function useDatabases(projectId: string): Database[] | undefined {
  const links = useSyncExternalStore(
    subscribe,
    () => store.get(projectId),
    () => undefined,
  );
  const planetscale = usePlanetscaleStore();
  return useMemo(() => {
    if (!links || !planetscale) {
      return undefined;
    }
    return links.map((link): Database => {
      const db = planetscale.databases.find(
        (d) => d.org === link.planetscale.org && d.name === link.name,
      );
      if (!db) {
        throw new Error(`${link.planetscale.org} has no database named ${link.name}.`);
      }
      return match(db.state)
        .with("provisioning", (): Database => ({ ...link, status: "provisioning" }))
        .with(
          "ready",
          (): Database => ({
            ...link,
            status: "ready",
            instance: {
              branchCount: db.branches.length,
              tableCount: db.tableCount,
              schemaUpdatedAt: db.schemaUpdatedAt,
              lastBackupAt: db.lastBackupAt,
              maintenanceAt: db.maintenanceAt,
              vitals: db.vitals,
            },
          }),
        )
        .exhaustive();
    });
  }, [links, planetscale]);
}

export function linkDatabase(projectId: string, input: CreateDatabaseInput): DatabaseLink {
  const branches = Object.values(input.branches).map((b) => b.branch);
  const db = match(input)
    .with({ source: "new" }, ({ org, name, engine, region, compute }) =>
      createPlanetscaleDatabase({ org, name, engine, region, ...compute, branches }),
    )
    .with({ source: "existing" }, ({ org, name }) =>
      addPlanetscaleBranches({ org, name }, branches),
    )
    .exhaustive();
  const link = linkOf(db, input.branches, {
    id: `db_${Math.random().toString(36).slice(2, 10)}`,
    createdAt: Date.now(),
  });
  setProject(projectId, (dbs) => [link, ...dbs]);
  return link;
}

export function removeDatabase(projectId: string, id: string) {
  setProject(projectId, (dbs) => dbs.filter((db) => db.id !== id));
}

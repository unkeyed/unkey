"use client";

import type { Loadable } from "@/lib/loadable/loadable";
import { useMutation } from "@tanstack/react-query";
import { useParams } from "next/navigation";
import { useEffect } from "react";
import type { CreateDatabaseInput, Database, DatabaseLink } from "./database-model";
import {
  PLANETSCALE_ORGS,
  connectPlanetscale,
  disconnectPlanetscale,
  usePlanetscaleStore,
} from "./planetscale-demo";
import {
  linkDatabase,
  removeDatabase,
  seedDatabases,
  useDatabases,
} from "./project-databases-demo";

function useProjectId(): string {
  return useParams<{ projectId: string }>().projectId;
}

export function useProjectDatabases(): { projectId: string; databases: Loadable<Database[]> } {
  const projectId = useProjectId();
  const databases = useDatabases(projectId);

  useEffect(() => {
    seedDatabases(projectId);
  }, [projectId]);

  return {
    projectId,
    databases:
      databases === undefined
        ? { status: "loading" }
        : { status: "ready", data: databases, stale: null },
  };
}

export function usePlanetscale() {
  const store = usePlanetscaleStore();
  return {
    connected: store?.connected ?? false,
    orgs: PLANETSCALE_ORGS,
    databases: store?.databases ?? [],
    connect: connectPlanetscale,
    disconnect: disconnectPlanetscale,
  };
}

export function useCreateDatabase() {
  const projectId = useProjectId();
  return useMutation({
    mutationFn: async (input: CreateDatabaseInput): Promise<DatabaseLink> =>
      linkDatabase(projectId, input),
  });
}

export function useDeleteDatabase() {
  const projectId = useProjectId();
  return useMutation({
    mutationFn: async (id: string) => removeDatabase(projectId, id),
  });
}

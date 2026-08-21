"use client";

import { trpc } from "@/lib/trpc/client";
import type { ResourceScope } from "../lib/catalogue.types";

export type ScopeInstance = {
  id: string;
  label: string;
  hint?: string;
};

export type ScopeInstances = {
  instances: ScopeInstance[];
  isLoading: boolean;
};

export function useScopeInstances(scope: ResourceScope): ScopeInstances {
  const keyspaces = trpc.deploy.environmentSettings.getAvailableKeyspaces.useQuery(undefined, {
    enabled: scope === "keyspaces",
  });
  const namespaces = trpc.ratelimit.namespace.list.useQuery(undefined, {
    enabled: scope === "ratelimit-namespaces",
  });

  switch (scope) {
    case "workspace":
      return { instances: [], isLoading: false };
    case "keyspaces":
      return {
        instances: Object.values(keyspaces.data ?? {}).map((keyspace) => ({
          id: keyspace.id,
          label: keyspace.api.name,
          hint: keyspace.id,
        })),
        isLoading: keyspaces.isLoading,
      };
    case "ratelimit-namespaces":
      return {
        instances: (namespaces.data ?? []).map((namespace) => ({
          id: namespace.id,
          label: namespace.name,
          hint: namespace.id,
        })),
        isLoading: namespaces.isLoading,
      };
  }
}

export function instanceLabels(instances: readonly ScopeInstance[]): Record<string, string> {
  return Object.fromEntries(instances.map((instance) => [instance.id, instance.label]));
}

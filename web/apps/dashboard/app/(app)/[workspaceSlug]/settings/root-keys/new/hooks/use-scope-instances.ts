"use client";

import { trpc } from "@/lib/trpc/client";
import type { ResourceScope } from "../lib/catalogue.types";
import { environmentLabel } from "../lib/policy";

export type ScopeInstance = {
  id: string;
  label: string;
  hint?: string;
};

export type ScopeInstances = {
  instances: ScopeInstance[];
  isLoading: boolean;
};

const DEPLOY_SCOPES: ResourceScope[] = ["projects", "apps", "environments"];

export function useScopeInstances(scope: ResourceScope): ScopeInstances {
  const projects = trpc.deploy.project.list.useQuery(undefined, {
    enabled: DEPLOY_SCOPES.includes(scope),
  });
  const environments = trpc.deploy.environment.listAll.useQuery(undefined, {
    enabled: scope === "environments",
  });
  const keyspaces = trpc.deploy.environmentSettings.getAvailableKeyspaces.useQuery(undefined, {
    enabled: scope === "keyspaces",
  });
  const namespaces = trpc.ratelimit.namespace.list.useQuery(undefined, {
    enabled: scope === "ratelimit-namespaces",
  });

  switch (scope) {
    case "workspace":
      return { instances: [], isLoading: false };
    case "projects":
      return {
        instances: (projects.data ?? []).map((project) => ({
          id: project.id,
          label: project.name,
          hint: project.id,
        })),
        isLoading: projects.isLoading,
      };
    case "apps":
      return {
        instances: (projects.data ?? []).flatMap((project) =>
          project.apps.map((app) => ({ id: app.id, label: app.name, hint: app.id })),
        ),
        isLoading: projects.isLoading,
      };
    case "environments": {
      const appNames = new Map(
        (projects.data ?? []).flatMap((project) =>
          project.apps.map((app) => [app.id, app.name] as const),
        ),
      );
      return {
        instances: (environments.data ?? []).map((environment) => ({
          id: environment.id,
          label: environmentLabel(appNames.get(environment.appId), environment.name),
          hint: environment.id,
        })),
        isLoading: environments.isLoading || projects.isLoading,
      };
    }
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

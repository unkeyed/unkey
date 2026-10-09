"use client";

import { useProjectEnvironments } from "@/hooks/use-project-environments";
import { useProjectsWithApps } from "@/hooks/use-projects-with-apps";
import { collection } from "@/lib/collections";
import { useEveryNamespace } from "@/lib/queries/ratelimit-namespaces";
import { trpc } from "@/lib/trpc/client";
import { eq, useLiveQuery } from "@tanstack/react-db";
import type { ResourceScope } from "../lib/catalogue.types";
import { environmentLabel } from "../lib/policy-view";

export type ScopeInstance = {
  id: string;
  label: string;
};

export type ScopeInstances = {
  instances: ScopeInstance[];
  isLoading: boolean;
};

const DEPLOY_SCOPES: ResourceScope[] = ["projects", "apps", "environments"];

export function useScopeInstances(scope: ResourceScope): ScopeInstances {
  const projects = useProjectsWithApps({
    enabled: DEPLOY_SCOPES.includes(scope),
  });
  const environments = useProjectEnvironments(scope === "environments" ? projects.data : undefined);
  const keyspaces = trpc.deploy.environmentSettings.getAvailableKeyspaces.useQuery(undefined, {
    enabled: scope === "keyspaces",
  });
  const namespaces = useEveryNamespace({ enabled: scope === "ratelimit-namespaces" });
  // Every namespace lives in the workspace default project
  const defaultProject = useLiveQuery(
    (q) =>
      scope === "ratelimit-namespaces"
        ? q
            .from({ project: collection.projects })
            .where(({ project }) => eq(project.isDefault, true))
        : null,
    [scope],
  );

  switch (scope) {
    case "workspace":
    case "identities":
    case "rbac":
      return { instances: [], isLoading: false };
    case "projects":
      return {
        instances: (projects.data ?? []).map((project) => ({
          id: project.id,
          label: project.name,
        })),
        isLoading: projects.isLoading,
      };
    case "apps":
      return {
        instances: (projects.data ?? []).flatMap((project) =>
          project.apps.map((app) => ({
            id: `projects/${project.id}/apps/${app.id}`,
            label: app.name,
          })),
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
          id: `projects/${environment.projectId}/apps/${environment.appId}/environments/${environment.id}`,
          label: environmentLabel(appNames.get(environment.appId), environment.slug),
        })),
        isLoading: environments.isLoading || projects.isLoading,
      };
    }
    case "keyspaces":
      return {
        instances: Object.values(keyspaces.data ?? {}).map((keyspace) => ({
          id: `projects/${keyspace.projectId}/keyspaces/${keyspace.id}`,
          label: keyspace.api.name,
        })),
        isLoading: keyspaces.isLoading,
      };
    case "ratelimit-namespaces": {
      const defaultProjectId = defaultProject.data?.at(0)?.id;
      return {
        instances: defaultProjectId
          ? (namespaces.data ?? []).map((namespace) => ({
              id: `projects/${defaultProjectId}/ratelimits/namespaces/${namespace.id}`,
              label: namespace.name,
            }))
          : [],
        isLoading: namespaces.isLoading || defaultProject.isLoading,
      };
    }
  }
}

export function instanceLabels(instances: readonly ScopeInstance[]): Record<string, string> {
  return Object.fromEntries(instances.map((instance) => [instance.id, instance.label]));
}

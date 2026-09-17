"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { useVisibleProjects } from "@/hooks/use-visible-projects";
import { isDeploymentInFlight } from "@/lib/collections/deploy/deployment-status";
import { routes } from "@/lib/navigation/routes";
import type { Route } from "next";
import { useMemo } from "react";

export type DeployState =
  | { kind: "loading" }
  | { kind: "no-apps"; href: Route }
  | { kind: "undeployed"; appCount: number; appName: string; projectName: string; href: Route }
  | {
      kind: "latest";
      appName: string;
      projectName: string;
      status: string;
      inFlight: boolean;
      deployedAt: number;
      href: Route;
    };

export function useDeployState(): DeployState {
  const workspace = useWorkspaceNavigation();
  const projects = useVisibleProjects();

  return useMemo(() => {
    if (projects.isLoading) {
      return { kind: "loading" };
    }

    const withApps = projects.data.flatMap((project) =>
      project.apps.map((app) => ({ project, app })),
    );

    if (withApps.length === 0) {
      return {
        kind: "no-apps",
        href: routes.projects.list({ workspaceSlug: workspace.slug, new: true }),
      };
    }

    const deployed = withApps.flatMap(({ project, app }) =>
      app.headlineDeployment ? [{ project, app, deployment: app.headlineDeployment }] : [],
    );

    if (deployed.length === 0) {
      const [first] = withApps;
      return {
        kind: "undeployed",
        appCount: withApps.length,
        appName: first.app.name,
        projectName: first.project.name,
        href: routes.projects.detail({
          workspaceSlug: workspace.slug,
          projectId: first.project.id,
        }),
      };
    }

    const latest = deployed.reduce((newest, candidate) =>
      candidate.deployment.deployedAt > newest.deployment.deployedAt ? candidate : newest,
    );

    return {
      kind: "latest",
      appName: latest.app.name,
      projectName: latest.project.name,
      status: latest.deployment.status,
      inFlight: isDeploymentInFlight(latest.deployment.status),
      deployedAt: latest.deployment.deployedAt,
      href: routes.projects.detail({
        workspaceSlug: workspace.slug,
        projectId: latest.project.id,
      }),
    };
  }, [projects.data, projects.isLoading, workspace.slug]);
}

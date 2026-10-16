"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { latestDeploymentId } from "@/lib/collections/deploy/apps";
import { routes } from "@/lib/navigation/routes";
import { useRouter } from "next/navigation";
import { type PropsWithChildren, useEffect } from "react";
import { useAppId, useProjectData } from "../data-provider";

export default function OverviewLayout({ children }: PropsWithChildren) {
  const router = useRouter();
  const workspace = useWorkspaceNavigation();
  const { projectId, apps } = useProjectData();
  const appId = useAppId();
  const app = apps.find((a) => a.id === appId);
  const neverDeployed = app !== undefined && latestDeploymentId(app) === null;
  const newAppHref = routes.projects.apps.new({ workspaceSlug: workspace.slug, projectId, appId });

  useEffect(() => {
    if (neverDeployed) {
      router.replace(newAppHref);
    }
  }, [neverDeployed, router, newAppHref]);

  return neverDeployed ? null : children;
}

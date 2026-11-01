"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { latestDeploymentId, useApp, useAppScope } from "@/lib/deploy-data";
import { routes } from "@/lib/navigation/routes";
import { useRouter } from "next/navigation";
import { type PropsWithChildren, useEffect } from "react";

export default function OverviewLayout({ children }: PropsWithChildren) {
  const router = useRouter();
  const workspace = useWorkspaceNavigation();
  const { projectId, appId } = useAppScope();
  const { data: app } = useApp();
  const neverDeployed = app !== undefined && latestDeploymentId(app) === null;
  const newAppHref = routes.projects.apps.new({ workspaceSlug: workspace.slug, projectId, appId });

  useEffect(() => {
    if (neverDeployed) {
      router.replace(newAppHref);
    }
  }, [neverDeployed, router, newAppHref]);

  return neverDeployed ? null : children;
}

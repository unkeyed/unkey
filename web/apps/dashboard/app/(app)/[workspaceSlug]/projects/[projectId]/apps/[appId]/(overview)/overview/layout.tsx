import { appOverview } from "@/lib/flags";
import { routes } from "@/lib/navigation/routes";
import { redirect } from "next/navigation";
import type { ReactNode } from "react";

export default async function OverviewLayout({
  children,
  params,
}: {
  children: ReactNode;
  params: Promise<{ workspaceSlug: string; projectId: string; appId: string }>;
}) {
  if (!(await appOverview())) {
    const { workspaceSlug, projectId, appId } = await params;
    redirect(routes.projects.apps.deployments({ workspaceSlug, projectId, appId }));
  }
  return children;
}

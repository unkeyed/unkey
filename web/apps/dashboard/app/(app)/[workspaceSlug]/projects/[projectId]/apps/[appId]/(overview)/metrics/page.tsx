"use client";

import { useFlag } from "@/lib/flags/provider";
import { routes } from "@/lib/navigation/routes";
import { redirect, useParams } from "next/navigation";
import { MetricsPage } from "./metrics-page";

export default function Page() {
  const enabled = useFlag("appMetrics");
  const { workspaceSlug, projectId, appId } = useParams<{
    workspaceSlug: string;
    projectId: string;
    appId: string;
  }>();
  if (!enabled) {
    redirect(routes.projects.apps.overview({ workspaceSlug, projectId, appId }));
  }
  return <MetricsPage />;
}

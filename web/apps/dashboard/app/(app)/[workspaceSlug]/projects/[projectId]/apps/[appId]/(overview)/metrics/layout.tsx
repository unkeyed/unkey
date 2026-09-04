import { appMetrics } from "@/lib/flags";
import { routes } from "@/lib/navigation/routes";
import { redirect } from "next/navigation";
import type { ReactNode } from "react";

// Server-side gate for the app Metrics tab. The flag defaults to off, so the
// page stays unreachable until app-metrics is enabled for the workspace or
// through FLAGS_LOCAL_OVERRIDES.
export default async function MetricsLayout({
  children,
  params,
}: {
  children: ReactNode;
  params: Promise<{ workspaceSlug: string; projectId: string; appId: string }>;
}) {
  if (!(await appMetrics())) {
    redirect(routes.projects.apps.overview(await params));
  }
  return children;
}

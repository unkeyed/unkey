"use client";

import { collection } from "@/lib/collections";
import { eq, useLiveQuery } from "@tanstack/react-db";
import { useProjectData } from "../../data-provider";

// The page heading is the app's name (the convention Vercel / Render / Supabase
// follow — "Overview" lives in the nav, not as the heading).
export function OverviewPageTitle() {
  const { projectId, appId } = useProjectData();

  const appsQuery = useLiveQuery(
    (q) => q.from({ app: collection.apps }).where(({ app }) => eq(app.projectId, projectId)),
    [projectId],
  );
  const appName = appsQuery.data?.find((a) => a.id === appId)?.name;

  return <>{appName ?? "App"}</>;
}

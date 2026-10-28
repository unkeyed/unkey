"use client";

import { collection } from "@/lib/collections";
import { eq, useLiveQuery } from "@tanstack/react-db";
import { Skeleton } from "@unkey/ui";
import { useProjectData } from "../../data-provider";

export function OverviewPageTitle() {
  const { projectId, appId } = useProjectData();

  const appsQuery = useLiveQuery(
    (q) => q.from({ app: collection.apps }).where(({ app }) => eq(app.projectId, projectId)),
    [projectId],
  );
  if (appsQuery.isLoading) {
    return <Skeleton className="inline-block h-6 w-32 align-middle" />;
  }
  const appName = appsQuery.data?.find((a) => a.id === appId)?.name;

  return <>{appName ?? "App"}</>;
}

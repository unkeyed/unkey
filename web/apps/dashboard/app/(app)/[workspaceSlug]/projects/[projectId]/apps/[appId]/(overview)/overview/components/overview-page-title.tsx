"use client";

import { useApp } from "@/lib/deploy-data";
import { Skeleton } from "@unkey/ui";

export function OverviewPageTitle() {
  const { data: app, isLoading } = useApp();
  if (isLoading) {
    return <Skeleton className="inline-block h-6 w-32 align-middle" />;
  }

  return <>{app?.name ?? "App"}</>;
}

"use client";

import { useProjectData } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/[appId]/(overview)/data-provider";
import { collection } from "@/lib/collections";
import { eq, useLiveQuery } from "@tanstack/react-db";
import { useMemo } from "react";

export function useAppFilterOptions(): AppFilterOption[] {
  const { projectId } = useProjectData();

  const apps = useLiveQuery(
    (q) => q.from({ app: collection.apps }).where(({ app }) => eq(app.projectId, projectId)),
    [projectId],
  );

  return useMemo(
    () =>
      (apps.data ?? []).map((app, i) => ({
        id: i,
        appId: app.id,
        name: app.name,
        checked: false,
      })),
    [apps.data],
  );
}

export function useAppNameById(): Map<string, string> {
  const options = useAppFilterOptions();

  return useMemo(() => new Map(options.map((option) => [option.appId, option.name])), [options]);
}

export function renderAppOption(option: AppFilterOption) {
  return <div className="text-accent-12 text-xs">{option.name}</div>;
}

type AppFilterOption = {
  id: number;
  appId: string;
  name: string;
  checked: boolean;
};

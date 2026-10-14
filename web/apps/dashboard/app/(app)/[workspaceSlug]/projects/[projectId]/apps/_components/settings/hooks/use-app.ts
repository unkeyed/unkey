"use client";

import { collection } from "@/lib/collections";
import { and, eq, useLiveQuery } from "@tanstack/react-db";
import { useAppId, useProjectData } from "../../../[appId]/(overview)/data-provider";

export function useApp() {
  const { projectId } = useProjectData();
  const appId = useAppId();
  const appQuery = useLiveQuery(
    (q) =>
      q
        .from({ app: collection.apps })
        .where(({ app }) => and(eq(app.projectId, projectId), eq(app.id, appId))),
    [projectId, appId],
  );
  return { projectId, appId, app: appQuery.data?.[0], isLoading: appQuery.isLoading };
}

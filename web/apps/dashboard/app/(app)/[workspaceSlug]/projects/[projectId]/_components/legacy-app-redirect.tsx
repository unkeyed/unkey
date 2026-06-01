"use client";
import { LoadingState } from "@/components/loading-state";
import { collection } from "@/lib/collections";
import { eq, useLiveQuery } from "@tanstack/react-db";
import { useParams, useRouter } from "next/navigation";
import { useEffect } from "react";

// Resolves a project's default app and redirects legacy project-scoped URLs
// (/projects/[projectId] and /projects/[projectId]/<tab>) to their app-scoped
// equivalent under /apps/[appId]. Keeps old bookmarks working after the move.
export function LegacyAppRedirect({ suffix }: { suffix: string }) {
  const params = useParams();
  const router = useRouter();
  const workspaceSlug = typeof params?.workspaceSlug === "string" ? params.workspaceSlug : "";
  const projectId = typeof params?.projectId === "string" ? params.projectId : "";

  const appsQuery = useLiveQuery(
    (q) => q.from({ app: collection.apps }).where(({ app }) => eq(app.projectId, projectId)),
    [projectId],
  );

  const apps = appsQuery.data ?? [];
  const defaultApp = apps.find((app) => app.slug === "default") ?? apps[0];

  useEffect(() => {
    if (!defaultApp) {
      return;
    }
    const base = `/${workspaceSlug}/projects/${projectId}/apps/${defaultApp.id}`;
    router.replace(suffix ? `${base}/${suffix}` : `${base}/deployments`);
  }, [defaultApp, router, workspaceSlug, projectId, suffix]);

  return <LoadingState message="Loading project..." />;
}

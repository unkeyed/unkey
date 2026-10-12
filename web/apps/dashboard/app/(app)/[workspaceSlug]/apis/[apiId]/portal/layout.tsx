import { getAuth } from "@/lib/auth/get-auth";
import { loadWorkspace } from "@/lib/db/load-workspace";
import { portalManagement } from "@/lib/flags";
import { routes } from "@/lib/navigation/routes";
import { resolveFlag } from "@/lib/workspace-flags";
import { redirect } from "next/navigation";
import type { ReactNode } from "react";
import { PortalNav } from "./components/portal-nav";

// Server-side gate mirroring the workspace-level portal layout: the flag
// defaults to off, so this route is unreachable until portal-management is
// enabled for the workspace or globally.
export default async function ApiPortalLayout({
  children,
  params,
}: {
  children: ReactNode;
  params: Promise<{ workspaceSlug: string; apiId: string; projectId?: string }>;
}) {
  const { orgId, userId } = await getAuth();
  const workspace = orgId && userId ? await loadWorkspace(orgId, [portalManagement]) : undefined;
  if (!resolveFlag({ workspace, flag: portalManagement })) {
    const { workspaceSlug, apiId, projectId } = await params;
    redirect(routes.apis.detail({ workspaceSlug, apiId, projectId }));
  }
  return <PortalNav>{children}</PortalNav>;
}

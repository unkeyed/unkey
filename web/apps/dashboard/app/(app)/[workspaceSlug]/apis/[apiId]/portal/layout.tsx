import { portalManagement } from "@/lib/flags";
import { routes } from "@/lib/navigation/routes";
import { redirect } from "next/navigation";
import type { ReactNode } from "react";

// Server-side gate mirroring the workspace-level portal layout: the flag
// defaults to off, so this route is unreachable until portal-management is
// enabled for the workspace or globally.
export default async function ApiPortalLayout({
  children,
  params,
}: {
  children: ReactNode;
  params: Promise<{ workspaceSlug: string; apiId: string }>;
}) {
  if (!(await portalManagement())) {
    const { workspaceSlug, apiId } = await params;
    redirect(routes.apis.detail({ workspaceSlug, apiId }));
  }
  return children;
}

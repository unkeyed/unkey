import { portalManagement } from "@/lib/flags";
import { routes } from "@/lib/navigation/routes";
import { redirect } from "next/navigation";
import type { ReactNode } from "react";

// Server-side gate for the portal configuration area. The flag defaults to off
// (see lib/flags/index.ts), so the page is unreachable until portal-management
// is enabled for the workspace or globally. Self-hosted dashboards without
// Vercel Flags get defaultValue: false unless overridden via FLAGS_LOCAL_OVERRIDES.
export default async function PortalLayout({
  children,
  params,
}: {
  children: ReactNode;
  params: Promise<{ workspaceSlug: string }>;
}) {
  if (!(await portalManagement())) {
    const { workspaceSlug } = await params;
    redirect(routes.projects.list({ workspaceSlug }));
  }
  return children;
}

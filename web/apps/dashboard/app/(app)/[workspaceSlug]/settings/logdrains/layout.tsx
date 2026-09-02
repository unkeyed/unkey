import { redirect } from "next/navigation";
import type { ReactNode } from "react";
import { logdrains } from "@/lib/flags";
import { routes } from "@/lib/navigation/routes";

// Server-side gate for the log drains settings area. The flag defaults to off
// so the pages are unreachable until logdrains is enabled for the workspace or
// globally. Self-hosted dashboards without Vercel Flags get
// defaultValue: false unless overridden via FLAGS_LOCAL_OVERRIDES.
export default async function LogdrainsLayout({
  children,
  params,
}: {
  children: ReactNode;
  params: Promise<{ workspaceSlug: string }>;
}) {
  if (!(await logdrains())) {
    const { workspaceSlug } = await params;
    redirect(routes.settings.general({ workspaceSlug }));
  }
  return children;
}

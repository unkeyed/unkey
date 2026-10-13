"use client";

import { useFlag } from "@/lib/flags/provider";
import { routes } from "@/lib/navigation/routes";
import { redirect, useParams } from "next/navigation";
import type { ReactNode } from "react";
import { PortalNav } from "./components/portal-nav";

export default function ApiPortalLayout({ children }: { children: ReactNode }) {
  const enabled = useFlag("portalManagement");
  const { workspaceSlug, apiId, projectId } = useParams<{
    workspaceSlug: string;
    apiId: string;
    projectId?: string;
  }>();
  if (!enabled) {
    redirect(routes.apis.detail({ workspaceSlug, apiId, projectId }));
  }
  return <PortalNav>{children}</PortalNav>;
}

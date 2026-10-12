"use client";

import { useApiKeyAuthId } from "@/hooks/use-api-key-auth-id";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { routes } from "@/lib/navigation/routes";
import { SecondaryNav, SecondaryNavGroup, SecondaryNavItem, SecondaryNavTitle } from "@unkey/ui";
import Link from "next/link";
import { useParams, useSelectedLayoutSegment } from "next/navigation";
import type { ReactNode } from "react";
import { usePortalSurfaceState } from "./portal-lifecycle-page";

export function PortalNav({ children }: { children: ReactNode }) {
  const workspace = useWorkspaceNavigation();
  const { apiId, projectId } = useParams<{ apiId: string; projectId?: string }>();
  const segment = useSelectedLayoutSegment();
  const {
    keyAuthId,
    isLoading: keyAuthIdLoading,
    isError: keyAuthIdError,
    refetch: refetchKeyAuthId,
  } = useApiKeyAuthId(apiId);
  const { state } = usePortalSurfaceState(
    keyAuthId,
    keyAuthIdLoading,
    keyAuthIdError,
    refetchKeyAuthId,
  );
  const scope = { workspaceSlug: workspace.slug, projectId, apiId };
  const items = [
    { key: "settings", label: "Settings", href: routes.apis.portal(scope), active: !segment },
    {
      key: "sessions",
      label: "Sessions",
      href: routes.apis.portalSessions(scope),
      active: segment === "sessions",
    },
  ];

  if (state.status === "notConfigured") {
    return children;
  }

  return (
    <div className="flex flex-col md:flex-row w-full flex-1 min-h-0">
      <SecondaryNav aria-label="Customer portal">
        <SecondaryNavTitle>Customer portal</SecondaryNavTitle>
        <SecondaryNavGroup>
          {items.map((item) => (
            <SecondaryNavItem
              key={item.key}
              active={item.active}
              render={<Link href={item.href} />}
            >
              {item.label}
            </SecondaryNavItem>
          ))}
        </SecondaryNavGroup>
      </SecondaryNav>
      <div className="flex-1 min-w-0">{children}</div>
    </div>
  );
}

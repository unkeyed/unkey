"use client";

import {
  RESOURCE_TYPE_PLURAL,
  RESOURCE_TYPE_ROUTES,
} from "@/components/navigation/sidebar/navigation-configs";
import { WorkspaceSwitcher } from "@/components/navigation/sidebar/team-switcher";
import { UserButton } from "@/components/navigation/sidebar/user-button";
import type { NavigationContext } from "@/hooks/use-navigation-context";
import { useNavigationContext } from "@/hooks/use-navigation-context";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { KEYSPACE_LABEL } from "@/lib/keyspace-label";
import { cn } from "@/lib/utils";
import Link from "next/link";
import { useSelectedLayoutSegments } from "next/navigation";
import { useMemo } from "react";

export const V3_HEADER_HEIGHT = 56;

type Tab = {
  id: string;
  label: string;
  href: string;
  active: boolean;
};

/**
 * v3 — header strip in the OG-Vercel mold.
 * Workspace switcher · workspace section tabs · user menu.
 * When inside a resource, a secondary tabs row appears underneath
 * carrying the resource's sub-nav (mirrors Vercel's project tabs).
 */
export function TopHeader() {
  const workspace = useWorkspaceNavigation();
  const context = useNavigationContext();
  const rawSegments = useSelectedLayoutSegments();
  const segments = useMemo(() => rawSegments ?? [], [rawSegments]);
  const base = `/${workspace.slug}`;

  const primaryTabs: Tab[] = [
    {
      id: "projects",
      label: "Projects",
      href: `${base}/projects`,
      active: segments.at(1) === "projects",
    },
    {
      id: "keyspaces",
      label: KEYSPACE_LABEL,
      href: `${base}/apis`,
      active: segments.at(1) === "apis",
    },
    {
      id: "ratelimit",
      label: "Ratelimit",
      href: `${base}/ratelimits`,
      active: segments.at(1) === "ratelimits",
    },
    {
      id: "authorization",
      label: "Authorization",
      href: `${base}/authorization/roles`,
      active: segments.includes("authorization"),
    },
    {
      id: "identities",
      label: "Identities",
      href: `${base}/identities`,
      active: segments.at(1) === "identities",
    },
    {
      id: "logs",
      label: "Logs",
      href: `${base}/logs`,
      active: segments.at(1) === "logs",
    },
    {
      id: "audit",
      label: "Audit",
      href: `${base}/audit`,
      active: segments.at(1) === "audit",
    },
    {
      id: "settings",
      label: "Settings",
      href: `${base}/settings/general`,
      active: segments.at(1) === "settings",
    },
  ];

  return (
    <header
      className="fixed inset-x-0 top-0 z-30 border-b border-grayA-4 bg-gray-1"
      style={{ height: V3_HEADER_HEIGHT }}
    >
      <div className="flex h-full items-center gap-4 px-4">
        <div className="shrink-0">
          <div className="[&_button]:!h-8 [&_button]:!w-[200px] [&_button]:!rounded-md [&_button]:!border [&_button]:!border-grayA-5 [&_button]:!px-2">
            <WorkspaceSwitcher />
          </div>
        </div>
        <nav className="flex flex-1 items-center gap-1 overflow-x-auto">
          {primaryTabs.map((tab) => (
            <Link
              key={tab.id}
              href={tab.href}
              className={cn(
                "whitespace-nowrap rounded-md px-2.5 py-1.5 text-[13px] font-medium transition-colors",
                tab.active
                  ? "bg-grayA-3 text-gray-12"
                  : "text-gray-11 hover:bg-grayA-2 hover:text-gray-12",
              )}
            >
              {tab.label}
            </Link>
          ))}
        </nav>
        <div className="shrink-0">
          <UserButton />
        </div>
      </div>
      {context.type === "resource" ? <SecondaryTabs context={context} /> : null}
    </header>
  );
}

/**
 * Sub-tab row that renders underneath the primary header when the user
 * is inside a resource (project / keyspace / namespace). Carries the
 * resource's nav config flat as horizontal tabs.
 */
function SecondaryTabs({
  context,
}: {
  context: Extract<NavigationContext, { type: "resource" }>;
}) {
  const workspace = useWorkspaceNavigation();
  const rawSegments = useSelectedLayoutSegments();
  const segments = useMemo(() => rawSegments ?? [], [rawSegments]);

  const items = useMemo<Tab[]>(() => {
    const wsBase = `/${workspace.slug}`;
    const resourceList = `${wsBase}/${RESOURCE_TYPE_ROUTES[context.resourceType]}`;
    const resourceBase = `${resourceList}/${context.resourceId}`;

    if (context.resourceType === "project") {
      return [
        {
          id: "deployments",
          label: "Deployments",
          href: `${resourceBase}/deployments`,
          active: segments.includes("deployments"),
        },
        {
          id: "logs",
          label: "Logs",
          href: `${resourceBase}/logs`,
          active: segments.includes("logs"),
        },
        {
          id: "requests",
          label: "Requests",
          href: `${resourceBase}/requests`,
          active: segments.includes("requests"),
        },
        {
          id: "env-vars",
          label: "Environment Variables",
          href: `${resourceBase}/env-vars`,
          active: segments.includes("env-vars"),
        },
        {
          id: "sentinel-policies",
          label: "Sentinel Policies",
          href: `${resourceBase}/sentinel-policies`,
          active: segments.includes("sentinel-policies"),
        },
        {
          id: "settings",
          label: "Settings",
          href: `${resourceBase}/settings`,
          active: segments.includes("settings"),
        },
      ];
    }

    if (context.resourceType === "api") {
      return [
        { id: "requests", label: "Requests", href: resourceBase, active: !segments.at(3) },
        {
          id: "keys",
          label: "Keys",
          href: `${resourceBase}/keys`,
          active: segments.includes("keys"),
        },
        {
          id: "settings",
          label: "Settings",
          href: `${resourceBase}/settings`,
          active: segments.includes("settings"),
        },
      ];
    }

    // namespace
    return [
      { id: "requests", label: "Requests", href: resourceBase, active: !segments.at(3) },
      {
        id: "logs",
        label: "Logs",
        href: `${resourceBase}/logs`,
        active: segments.includes("logs"),
      },
      {
        id: "settings",
        label: "Settings",
        href: `${resourceBase}/settings`,
        active: segments.includes("settings"),
      },
      {
        id: "overrides",
        label: "Overrides",
        href: `${resourceBase}/overrides`,
        active: segments.includes("overrides"),
      },
    ];
  }, [context, segments, workspace.slug]);

  const plural = RESOURCE_TYPE_PLURAL[context.resourceType];
  const listHref = `/${workspace.slug}/${RESOURCE_TYPE_ROUTES[context.resourceType]}`;

  return (
    <div className="absolute inset-x-0 top-full border-b border-grayA-4 bg-gray-1">
      <div className="flex items-center gap-1 px-4 py-1.5">
        <Link
          href={listHref}
          className="mr-2 text-[12px] font-medium text-gray-11 hover:text-gray-12"
        >
          ← All {plural}
        </Link>
        <span className="mr-2 h-4 w-px bg-grayA-4" aria-hidden />
        {items.map((tab) => (
          <Link
            key={tab.id}
            href={tab.href}
            className={cn(
              "whitespace-nowrap rounded-md px-2.5 py-1 text-[12px] font-medium transition-colors",
              tab.active
                ? "bg-grayA-3 text-gray-12"
                : "text-gray-11 hover:bg-grayA-2 hover:text-gray-12",
            )}
          >
            {tab.label}
          </Link>
        ))}
      </div>
    </div>
  );
}

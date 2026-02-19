import type { Workspace } from "@/lib/db";
import {
  Cube,
  Earth,
  Fingerprint,
  Gauge,
  Gear,
  Grid,
  InputSearch,
  Key,
  Layers3,
  Nodes,
  Nodes2,
  ShieldKey,
  Sparkle3,
} from "@unkey/icons";
import { cn } from "../../../lib/utils";
import type { NavItem } from "./workspace-navigations";

const Tag: React.FC<{ label: string; className?: string }> = ({ label, className }) => (
  <div
    className={cn(
      "border text-gray-11 border-gray-6 hover:border-gray-8 rounded text-xs px-1 py-0.5 font-mono",
      className,
    )}
  >
    {label}
  </div>
);

/**
 * API Management Product Navigation (product-level)
 */
export function createApiManagementNavigation(segments: string[], workspace: Workspace): NavItem[] {
  const basePath = `/${workspace.slug}`;

  return [
    {
      icon: Nodes,
      href: `${basePath}/apis`,
      label: "APIs",
      active: segments.at(1) === "apis" && !segments.at(2), // Only active at list level
    },
    {
      icon: Gauge,
      href: `${basePath}/ratelimits`,
      label: "Ratelimit",
      active: segments.at(1) === "ratelimits" && !segments.at(2), // Only active at list level
    },
    {
      icon: ShieldKey,
      label: "Authorization",
      href: `${basePath}/authorization/roles`,
      active: segments.some((s) => s === "authorization"),
      items: [
        {
          icon: null,
          label: "Roles",
          href: `${basePath}/authorization/roles`,
          active: segments.some((s) => s === "roles"),
        },
        {
          icon: null,
          label: "Permissions",
          href: `${basePath}/authorization/permissions`,
          active: segments.some((s) => s === "permissions"),
        },
      ],
    },
    {
      icon: Grid,
      href: "/monitors/verifications",
      label: "Monitors",
      active: segments.at(1) === "verifications",
      hidden: !workspace?.features.webhooks,
    },
    {
      icon: Layers3,
      href: `${basePath}/logs`,
      label: "Logs",
      active: segments.at(1) === "logs",
    },
    {
      icon: Fingerprint,
      href: `${basePath}/identities`,
      label: "Identities",
      active: segments.at(1) === "identities" && !segments.at(2), // Only active at list level
    },
  ].filter((n) => !n.hidden);
}

/**
 * Deploy Product Navigation (product-level)
 */
export function createDeployNavigation(segments: string[], workspace: Workspace): NavItem[] {
  const basePath = `/${workspace.slug}`;

  return [
    {
      icon: Cube,
      href: `${basePath}/projects`,
      label: "Projects",
      active: segments.at(1) === "projects" && !segments.at(2), // Only active at list level
      tag: <Tag label="Beta" className="mr-2 group-hover:bg-gray-1" />,
    },
    {
      icon: Earth,
      href: `${basePath}/domains`,
      label: "Domains",
      active: segments.at(1) === "domains",
    },
    {
      icon: Nodes2,
      href: `${basePath}/environment-variables`,
      label: "Environment Variables",
      active: segments.at(1) === "environment-variables",
    },
  ];
}

/**
 * Specific API Navigation (resource-level)
 */
export function createApiNavigation(
  apiId: string,
  workspace: Workspace,
  segments: string[],
): NavItem[] {
  const basePath = `/${workspace.slug}/apis/${apiId}`;

  return [
    {
      icon: Key,
      href: `${basePath}/keys`,
      label: "Keys",
      active: segments.includes("keys"),
    },
    {
      icon: Gear,
      href: `${basePath}/settings`,
      label: "Settings",
      active: segments.includes("settings") && segments.includes("apis"),
    },
  ];
}

/**
 * Specific Project Navigation (resource-level)
 */
export function createProjectNavigation(
  projectId: string,
  workspace: Workspace,
  segments: string[],
): NavItem[] {
  const basePath = `/${workspace.slug}/projects/${projectId}`;

  return [
    {
      icon: Grid,
      href: basePath,
      label: "Overview",
      active: segments.at(2) === projectId && !segments.at(3),
    },
    {
      icon: Cube,
      href: `${basePath}/deployments`,
      label: "Deployments",
      active: segments.includes("deployments"),
    },
    {
      icon: Layers3,
      href: `${basePath}/logs`,
      label: "Logs",
      active: segments.includes("logs") && segments.includes("projects"),
    },
    {
      icon: InputSearch,
      href: `${basePath}/requests`,
      label: "Request Logs",
      active: segments.includes("requests"),
    },
    {
      icon: Gear,
      href: `${basePath}/settings`,
      label: "Settings",
      active: segments.includes("settings") && segments.includes("projects"),
    },
  ];
}

/**
 * Specific Namespace Navigation (resource-level)
 */
export function createNamespaceNavigation(
  namespaceId: string,
  workspace: Workspace,
  segments: string[],
): NavItem[] {
  const basePath = `/${workspace.slug}/ratelimits/${namespaceId}`;

  return [
    {
      icon: Grid,
      href: basePath,
      label: "Overview",
      active: segments.at(2) === namespaceId && !segments.at(3),
    },
    {
      icon: Layers3,
      href: `${basePath}/overrides`,
      label: "Overrides",
      active: segments.includes("overrides"),
    },
    {
      icon: Sparkle3,
      href: `${basePath}/logs`,
      label: "Analytics",
      active: segments.includes("logs") && segments.includes("ratelimits"),
    },
    {
      icon: Gear,
      href: `${basePath}/settings`,
      label: "Settings",
      active: segments.includes("settings") && segments.includes("ratelimits"),
    },
  ];
}

import type { Workspace } from "@/lib/db";
import {
  ArrowDottedRotateAnticlockwise,
  ArrowOppositeDirectionY,
  Cube,
  Earth,
  Fingerprint,
  Gauge,
  Gear,
  Grid,
  Key,
  Layers3,
  Nodes,
  Nodes2,
  ShieldKey,
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
 * The Keys link requires a keyAuthId (keyspace ID).
 * - If keyAuthId is provided (we're on a keys page), use it directly
 * - Otherwise, add a placeholder that useApiKeyspace hook will enhance by fetching the API data
 */
export function createApiNavigation(
  apiId: string,
  workspace: Workspace,
  segments: string[],
  keyAuthId?: string,
): NavItem[] {
  const basePath = `/${workspace.slug}/apis/${apiId}`;

  const childItems: NavItem[] = [];

  // Add Overview/Requests link - this is the main API page
  childItems.push({
    icon: ArrowOppositeDirectionY,
    href: basePath,
    label: "Requests",
    active: segments.at(2) === apiId && !segments.at(3),
  });

  // Add Keys link
  if (keyAuthId) {
    // We have the keyAuthId from URL params, use it directly
    childItems.push({
      icon: Key,
      href: `${basePath}/keys/${keyAuthId}`,
      label: "Keys",
      active: segments.includes("keys"),
    });
  } else {
    // Add placeholder that will be enhanced by useApiKeyspace hook
    // The hook will fetch the API data and update this with the correct keyAuthId
    childItems.push({
      icon: Key,
      href: `${basePath}`, // Temporary href, will be updated by hook
      label: "Keys",
      active: segments.includes("keys"),
      disabled: true, // Mark as disabled until hook updates it
    });
  }

  childItems.push({
    icon: Gear,
    href: `${basePath}/settings`,
    label: "Settings",
    active: segments.includes("settings") && segments.includes("apis"),
  });

  // Return as a single parent item with children
  return [
    {
      icon: Layers3,
      href: basePath,
      label: apiId, // Will be replaced with actual API name by hook
      active: segments.includes(apiId),
      items: childItems,
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

  const childItems: NavItem[] = [
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
      icon: ArrowOppositeDirectionY,
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

  return [
    {
      icon: Cube,
      href: basePath,
      label: projectId, // Will be replaced with actual project name
      active: segments.includes(projectId),
      items: childItems,
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

  const childItems: NavItem[] = [
    {
      icon: ArrowOppositeDirectionY,
      href: basePath,
      label: "Requests",
      active: segments.at(2) === namespaceId && !segments.at(3),
    },
    {
      icon: Layers3,
      href: `${basePath}/logs`,
      label: "Logs",
      active: segments.includes("logs") && segments.includes("ratelimits"),
    },
    {
      icon: Gear,
      href: `${basePath}/settings`,
      label: "Settings",
      active: segments.includes("settings") && segments.includes("ratelimits"),
    },
    {
      icon: ArrowDottedRotateAnticlockwise,
      href: `${basePath}/overrides`,
      label: "Overrides",
      active: segments.includes("overrides"),
    },
  ];

  return [
    {
      icon: Gauge,
      href: basePath,
      label: namespaceId, // Will be replaced with actual namespace name
      active: segments.includes(namespaceId),
      items: childItems,
    },
  ];
}

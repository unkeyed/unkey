import {
  ArrowDottedRotateAnticlockwise,
  ArrowOppositeDirectionY,
  BracketsSquareDots,
  Cube,
  Fingerprint,
  Gauge,
  Gear,
  InputSearch,
  Key,
  Layers3,
  Nodes,
  ShieldKey,
  SquareBulletList,
} from "@unkey/icons";
import { appPath, projectLogsPath, projectPath, projectRequestsPath, projectsPath } from "./routes";
import type { ResolvedNavLink } from "./types";

export function buildWorkspaceSections(slug: string, segments: string[]): ResolvedNavLink[] {
  const top = segments[0];
  return [
    {
      key: "projects",
      label: "Projects",
      href: projectsPath({ workspaceSlug: slug }),
      icon: Cube,
      isActive: top === "projects",
    },
    {
      key: "apis",
      label: "Keyspaces (APIs)",
      href: `/${slug}/apis`,
      icon: Nodes,
      isActive: top === "apis",
    },
    {
      key: "ratelimits",
      label: "Ratelimit",
      href: `/${slug}/ratelimits`,
      icon: Gauge,
      isActive: top === "ratelimits",
    },
    {
      key: "authorization",
      label: "Authorization",
      href: `/${slug}/authorization/roles`,
      icon: ShieldKey,
      isActive: top === "authorization",
    },
    {
      key: "logs",
      label: "Logs",
      href: `/${slug}/logs`,
      icon: Layers3,
      isActive: top === "logs",
    },
    {
      key: "identities",
      label: "Identities",
      href: `/${slug}/identities`,
      icon: Fingerprint,
      isActive: top === "identities",
    },
    {
      key: "audit",
      label: "Audit Log",
      href: `/${slug}/audit`,
      icon: InputSearch,
      isActive: top === "audit",
    },
    {
      key: "settings",
      label: "Settings",
      href: `/${slug}/settings/general`,
      icon: Gear,
      isActive: top === "settings",
    },
  ];
}

export function buildSettingsLinks(slug: string, segments: string[]): ResolvedNavLink[] {
  const page = segments[1];
  const base = `/${slug}/settings`;
  return [
    { key: "general", label: "General", href: `${base}/general`, isActive: page === "general" },
    { key: "team", label: "Team", href: `${base}/team`, isActive: page === "team" },
    {
      key: "root-keys",
      label: "Root Keys",
      href: `${base}/root-keys`,
      isActive: page === "root-keys",
    },
    { key: "billing", label: "Billing", href: `${base}/billing`, isActive: page === "billing" },
  ];
}

export function buildAuthorizationLinks(slug: string, segments: string[]): ResolvedNavLink[] {
  const page = segments[1];
  const base = `/${slug}/authorization`;
  return [
    { key: "roles", label: "Roles", href: `${base}/roles`, isActive: page === "roles" },
    {
      key: "permissions",
      label: "Permissions",
      href: `${base}/permissions`,
      isActive: page === "permissions",
    },
  ];
}

export function buildProjectLinks(
  slug: string,
  projectId: string,
  segments: string[],
): ResolvedNavLink[] {
  const page = segments[2];
  const base = projectPath({ workspaceSlug: slug, projectId });
  return [
    {
      key: "overview",
      label: "Overview",
      href: base,
      icon: Cube,
      isActive: !page,
    },
    {
      key: "logs",
      label: "Logs",
      href: `${base}/logs`,
      icon: Layers3,
      isActive: page === "logs",
    },
    {
      key: "requests",
      label: "Requests",
      href: `${base}/requests`,
      icon: ArrowOppositeDirectionY,
      isActive: page === "requests",
    },
    {
      key: "settings",
      label: "Settings",
      href: `${base}/settings`,
      icon: Gear,
      isActive: page === "settings",
    },
  ];
}

export function buildAppLinks(
  slug: string,
  projectId: string,
  appId: string,
  segments: string[],
): ResolvedNavLink[] {
  const page = segments[4];
  const base = appPath({ workspaceSlug: slug, projectId, appId });
  return [
    {
      key: "deployments",
      label: "Deployments",
      href: `${base}/deployments`,
      icon: SquareBulletList,
      isActive: page === "deployments",
    },
    {
      key: "env-vars",
      label: "Environment Variables",
      href: `${base}/env-vars`,
      icon: BracketsSquareDots,
      isActive: page === "env-vars",
    },
    {
      key: "sentinel-policies",
      label: "Sentinel Policies",
      href: `${base}/sentinel-policies`,
      icon: ShieldKey,
      isActive: page === "sentinel-policies",
    },
    {
      key: "settings",
      label: "Settings",
      href: `${base}/settings`,
      icon: Gear,
      isActive: page === "settings",
    },
    // Project-level views scoped to this app; separated since they navigate
    // out of the app section.
    {
      key: "logs",
      label: "Logs",
      href: projectLogsPath({ workspaceSlug: slug, projectId, appId }),
      icon: Layers3,
      isActive: false,
      separatorAbove: true,
    },
    {
      key: "requests",
      label: "Requests",
      href: projectRequestsPath({ workspaceSlug: slug, projectId, since: "6h", appId }),
      icon: ArrowOppositeDirectionY,
      isActive: false,
    },
    // Will be polished and added back in the future iterations
    // {
    //   key: "openapi-diff",
    //   label: "OpenAPI Diff",
    //   href: `${base}/openapi-diff`,
    //   icon: Nodes,
    //   isActive: page === "openapi-diff",
    // },
  ];
}

export function buildApiLinks(
  slug: string,
  apiId: string,
  keyAuthId: string | undefined,
  segments: string[],
): ResolvedNavLink[] {
  const page = segments[2];
  const base = `/${slug}/apis/${apiId}`;
  return [
    {
      key: "requests",
      label: "Requests",
      href: base,
      icon: ArrowOppositeDirectionY,
      isActive: !page,
    },
    {
      key: "keys",
      label: "Keys",
      href: keyAuthId ? `${base}/keys/${keyAuthId}` : base,
      icon: Key,
      isActive: page === "keys",
      disabled: !keyAuthId,
    },
    {
      key: "settings",
      label: "Settings",
      href: `${base}/settings`,
      icon: Gear,
      isActive: page === "settings",
    },
  ];
}

export function buildNamespaceLinks(
  slug: string,
  namespaceId: string,
  segments: string[],
): ResolvedNavLink[] {
  const page = segments[2];
  const base = `/${slug}/ratelimits/${namespaceId}`;
  return [
    {
      key: "requests",
      label: "Requests",
      href: base,
      icon: ArrowOppositeDirectionY,
      isActive: !page,
    },
    {
      key: "logs",
      label: "Logs",
      href: `${base}/logs`,
      icon: Layers3,
      isActive: page === "logs",
    },
    {
      key: "settings",
      label: "Settings",
      href: `${base}/settings`,
      icon: Gear,
      isActive: page === "settings",
    },
    {
      key: "overrides",
      label: "Overrides",
      href: `${base}/overrides`,
      icon: ArrowDottedRotateAnticlockwise,
      isActive: page === "overrides",
    },
  ];
}

import {
  ArrowDottedRotateAnticlockwise,
  ArrowOppositeDirectionY,
  BracketsSquareDots,
  Cube,
  Fingerprint,
  Gauge,
  Gear,
  Key,
  Layers3,
  Nodes,
  ShieldKey,
} from "@unkey/icons";
import type { ResolvedNavLink } from "./types";

export function buildWorkspaceSections(slug: string, segments: string[]): ResolvedNavLink[] {
  const top = segments[0];
  return [
    {
      key: "projects",
      label: "Projects",
      href: `/${slug}/projects`,
      icon: Cube,
      isActive: top === "projects",
    },
    {
      key: "apis",
      label: "APIs",
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
  const base = `/${slug}/projects/${projectId}`;
  return [
    {
      key: "deployments",
      label: "Deployments",
      href: `${base}/deployments`,
      icon: Cube,
      isActive: page === "deployments",
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

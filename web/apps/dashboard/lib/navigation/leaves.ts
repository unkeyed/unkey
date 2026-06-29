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
import { routes } from "./routes";
import type { ResolvedNavLink } from "./types";

export function buildWorkspaceSections(slug: string, segments: string[]): ResolvedNavLink[] {
  const top = segments[0];
  return [
    {
      key: "projects",
      label: "Projects",
      href: routes.projects.list({ workspaceSlug: slug }),
      icon: Cube,
      isActive: top === "projects",
    },
    {
      key: "apis",
      label: "Keyspaces (APIs)",
      href: routes.apis.list({ workspaceSlug: slug }),
      icon: Nodes,
      isActive: top === "apis",
    },
    {
      key: "ratelimits",
      label: "Ratelimit",
      href: routes.ratelimits.list({ workspaceSlug: slug }),
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
      href: routes.settings.general({ workspaceSlug: slug }),
      icon: Gear,
      isActive: top === "settings",
    },
  ];
}

export function buildProjectLinks(
  slug: string,
  projectId: string,
  segments: string[],
): ResolvedNavLink[] {
  const page = segments[2];
  const scope = { workspaceSlug: slug, projectId };
  return [
    {
      key: "apps",
      label: "Apps",
      href: routes.projects.detail(scope),
      icon: Cube,
      isActive: !page,
    },
    {
      key: "logs",
      label: "Logs",
      href: routes.projects.logs(scope),
      icon: Layers3,
      isActive: page === "logs",
    },
    {
      key: "requests",
      label: "Requests",
      href: routes.projects.requests(scope),
      icon: ArrowOppositeDirectionY,
      isActive: page === "requests",
    },
    {
      key: "settings",
      label: "Project Settings",
      href: routes.projects.settings(scope),
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
  appOverviewEnabled: boolean,
): ResolvedNavLink[] {
  const page = segments[4];
  const scope = { workspaceSlug: slug, projectId, appId };
  const overviewLink: ResolvedNavLink = {
    key: "overview",
    label: "Overview",
    href: routes.projects.apps.overview(scope),
    icon: Cube,
    isActive: page === "overview",
  };
  const legacyLinks: ResolvedNavLink[] = [
    {
      key: "logs",
      label: "Logs",
      href: routes.projects.logs(scope),
      icon: Layers3,
      isActive: false,
      separatorAbove: true,
    },
    {
      key: "requests",
      label: "Requests",
      href: routes.projects.requests({ ...scope, since: "6h" }),
      icon: ArrowOppositeDirectionY,
      isActive: false,
    },
  ];
  return [
    ...(appOverviewEnabled ? [overviewLink] : []),
    {
      key: "deployments",
      label: "Deployments",
      href: routes.projects.apps.deployments(scope),
      icon: SquareBulletList,
      isActive: page === "deployments",
    },
    {
      key: "env-vars",
      label: "Environment Variables",
      href: routes.projects.apps.envVars(scope),
      icon: BracketsSquareDots,
      isActive: page === "env-vars",
    },
    {
      key: "sentinel-policies",
      label: "Sentinel Policies",
      href: routes.projects.apps.sentinelPolicies(scope),
      icon: ShieldKey,
      isActive: page === "sentinel-policies",
    },
    {
      key: "settings",
      label: "App Settings",
      href: routes.projects.apps.settings(scope),
      icon: Gear,
      isActive: page === "settings",
    },
    ...(appOverviewEnabled ? [] : legacyLinks),
    // Will be polished and added back in the future iterations
    // {
    //   key: "openapi-diff",
    //   label: "OpenAPI Diff",
    //   href: routes.projects.apps.openapiDiff(...),
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
  return [
    {
      key: "requests",
      label: "Requests",
      href: routes.apis.detail({ workspaceSlug: slug, apiId }),
      icon: ArrowOppositeDirectionY,
      isActive: !page,
    },
    {
      key: "keys",
      label: "Keys",
      href: keyAuthId
        ? routes.apis.keys.list({ workspaceSlug: slug, apiId, keyAuthId })
        : routes.apis.detail({ workspaceSlug: slug, apiId }),
      icon: Key,
      isActive: page === "keys",
      disabled: !keyAuthId,
    },
    {
      key: "settings",
      label: "Settings",
      href: routes.apis.settings({ workspaceSlug: slug, apiId }),
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
  const scope = { workspaceSlug: slug, namespaceId };
  return [
    {
      key: "requests",
      label: "Requests",
      href: routes.ratelimits.detail(scope),
      icon: ArrowOppositeDirectionY,
      isActive: !page,
    },
    {
      key: "logs",
      label: "Logs",
      href: routes.ratelimits.logs(scope),
      icon: Layers3,
      isActive: page === "logs",
    },
    {
      key: "settings",
      label: "Settings",
      href: routes.ratelimits.settings(scope),
      icon: Gear,
      isActive: page === "settings",
    },
    {
      key: "overrides",
      label: "Overrides",
      href: routes.ratelimits.overrides(scope),
      icon: ArrowDottedRotateAnticlockwise,
      isActive: page === "overrides",
    },
  ];
}

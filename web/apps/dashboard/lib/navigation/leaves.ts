import {
  IconArrowDottedRotateAnticlockwiseOutline18,
  IconArrowsOppositeDirectionYOutline18,
  IconBracketsSquareDotsOutline18,
  IconCubeOutline18,
  IconFingerprintOutline18,
  IconGaugeOutline18,
  IconGearOutline18,
  IconInputSearchOutline18,
  IconKeyOutline18,
  IconLayers3Outline18,
  IconNodesOutline18,
  IconShieldKeyOutline18,
  IconSquareBulletListOutline18,
  IconWindowLayoutOutline18,
} from "nucleo-ui-outline-18";
import { routes } from "./routes";
import type { ResolvedNavLink } from "./types";

export function buildWorkspaceSections(slug: string, segments: string[]): ResolvedNavLink[] {
  const top = segments[0];
  return [
    {
      key: "projects",
      label: "Projects",
      href: routes.projects.list({ workspaceSlug: slug }),
      icon: IconCubeOutline18,
      isActive: top === "projects",
    },
    {
      key: "apis",
      label: "Keyspaces (APIs)",
      href: routes.apis.list({ workspaceSlug: slug }),
      icon: IconNodesOutline18,
      isActive: top === "apis",
    },
    {
      key: "ratelimits",
      label: "Ratelimit",
      href: routes.ratelimits.list({ workspaceSlug: slug }),
      icon: IconGaugeOutline18,
      isActive: top === "ratelimits",
    },
    {
      key: "authorization",
      label: "Authorization",
      href: `/${slug}/authorization/roles`,
      icon: IconShieldKeyOutline18,
      isActive: top === "authorization",
    },
    {
      key: "logs",
      label: "Logs",
      href: `/${slug}/logs`,
      icon: IconLayers3Outline18,
      isActive: top === "logs",
    },
    {
      key: "identities",
      label: "Identities",
      href: `/${slug}/identities`,
      icon: IconFingerprintOutline18,
      isActive: top === "identities",
    },
    {
      key: "audit",
      label: "Audit Log",
      href: `/${slug}/audit`,
      icon: IconInputSearchOutline18,
      isActive: top === "audit",
    },
    {
      key: "settings",
      label: "Settings",
      href: routes.settings.general({ workspaceSlug: slug }),
      icon: IconGearOutline18,
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
      icon: IconCubeOutline18,
      isActive: !page,
    },
    {
      key: "logs",
      label: "Logs",
      href: routes.projects.logs(scope),
      icon: IconLayers3Outline18,
      isActive: page === "logs",
    },
    {
      key: "requests",
      label: "Requests",
      href: routes.projects.requests(scope),
      icon: IconArrowsOppositeDirectionYOutline18,
      isActive: page === "requests",
    },
    {
      key: "settings",
      label: "Project Settings",
      href: routes.projects.settings(scope),
      icon: IconGearOutline18,
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
  const scope = { workspaceSlug: slug, projectId, appId };
  return [
    {
      key: "overview",
      label: "Overview",
      href: routes.projects.apps.overview(scope),
      icon: IconCubeOutline18,
      isActive: page === "overview",
    },
    {
      key: "deployments",
      label: "Deployments",
      href: routes.projects.apps.deployments(scope),
      icon: IconSquareBulletListOutline18,
      isActive: page === "deployments",
    },
    {
      key: "env-vars",
      label: "Environment Variables",
      href: routes.projects.apps.envVars(scope),
      icon: IconBracketsSquareDotsOutline18,
      isActive: page === "env-vars",
    },
    {
      key: "policies",
      label: "Policies",
      href: routes.projects.apps.policies(scope),
      icon: IconShieldKeyOutline18,
      isActive: page === "policies",
    },
    {
      key: "settings",
      label: "App Settings",
      href: routes.projects.apps.settings(scope),
      icon: IconGearOutline18,
      isActive: page === "settings",
    },
    {
      key: "logs",
      label: "Go to Logs",
      href: routes.projects.logs(scope),
      icon: IconLayers3Outline18,
      isActive: page === "logs",
      separatorAbove: true,
    },
    {
      key: "requests",
      label: "Go to Requests",
      href: routes.projects.requests(scope),
      icon: IconArrowsOppositeDirectionYOutline18,
      isActive: page === "requests",
    },
    // Will be polished and added back in the future iterations
    // {
    //   key: "openapi-diff",
    //   label: "OpenAPI Diff",
    //   href: routes.projects.apps.openapiDiff(...),
    //   icon: IconNodesOutline18,
    //   isActive: page === "openapi-diff",
    // },
  ];
}

export function buildApiLinks(
  slug: string,
  apiId: string,
  keyAuthId: string | undefined,
  segments: string[],
  portalManagementEnabled: boolean,
): ResolvedNavLink[] {
  const page = segments[2];
  const portalLink: ResolvedNavLink = {
    key: "portal",
    label: "Customer portal",
    href: routes.apis.portal({ workspaceSlug: slug, apiId }),
    icon: IconWindowLayoutOutline18,
    isActive: page === "portal",
  };
  return [
    {
      key: "requests",
      label: "Requests",
      href: routes.apis.detail({ workspaceSlug: slug, apiId }),
      icon: IconArrowsOppositeDirectionYOutline18,
      isActive: !page,
    },
    {
      key: "keys",
      label: "Keys",
      href: keyAuthId
        ? routes.apis.keys.list({ workspaceSlug: slug, apiId, keyAuthId })
        : routes.apis.detail({ workspaceSlug: slug, apiId }),
      icon: IconKeyOutline18,
      isActive: page === "keys",
      disabled: !keyAuthId,
    },
    ...(portalManagementEnabled ? [portalLink] : []),
    {
      key: "settings",
      label: "Settings",
      href: routes.apis.settings({ workspaceSlug: slug, apiId }),
      icon: IconGearOutline18,
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
      icon: IconArrowsOppositeDirectionYOutline18,
      isActive: !page,
    },
    {
      key: "logs",
      label: "Logs",
      href: routes.ratelimits.logs(scope),
      icon: IconLayers3Outline18,
      isActive: page === "logs",
    },
    {
      key: "settings",
      label: "Settings",
      href: routes.ratelimits.settings(scope),
      icon: IconGearOutline18,
      isActive: page === "settings",
    },
    {
      key: "overrides",
      label: "Overrides",
      href: routes.ratelimits.overrides(scope),
      icon: IconArrowDottedRotateAnticlockwiseOutline18,
      isActive: page === "overrides",
    },
  ];
}

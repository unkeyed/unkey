// Kept apart from ./leaves.ts so deleting the flag deletes that file whole.
import {
  IconArrowsOppositeDirectionYOutline18,
  IconCubeOutline18,
  IconFingerprintOutline18,
  IconGaugeOutline18,
  IconGearOutline18,
  IconInputSearchOutline18,
  IconKeyOutline18,
  IconLayers3Outline18,
  IconNodesOutline18,
  IconShieldKeyOutline18,
} from "@unkey/icons";
import { routes } from "./routes";
import type { ResolvedNavLink } from "./types";

export function buildWorkspaceSections(
  slug: string,
  segments: string[],
  canSeeRootKeys: boolean,
): ResolvedNavLink[] {
  const top = segments[0];
  const rootKeys: ResolvedNavLink[] = canSeeRootKeys
    ? [
        {
          key: "root-keys",
          label: "Root Keys",
          href: routes.rootKeys.list({ workspaceSlug: slug }),
          icon: IconKeyOutline18,
          isActive: top === "root-keys",
        },
      ]
    : [];
  return [
    {
      key: "projects",
      label: "Projects",
      href: routes.projects.list({ workspaceSlug: slug }),
      icon: IconCubeOutline18,
      isActive: top === "projects",
    },
    ...rootKeys,
    {
      key: "logs",
      label: "Logs",
      href: routes.logs.list({ workspaceSlug: slug }),
      icon: IconLayers3Outline18,
      isActive: top === "logs",
    },
    {
      key: "audit",
      label: "Audit Log",
      href: routes.audit.list({ workspaceSlug: slug }),
      icon: IconInputSearchOutline18,
      isActive: top === "audit",
    },
    {
      key: "settings",
      label: "Workspace Settings",
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
  // isDefault is undefined until the project row has loaded; the deploy links wait for it.
  { isDefault }: { isDefault: boolean | undefined },
): ResolvedNavLink[] {
  const page = segments[2];
  const scope = { workspaceSlug: slug, projectId };
  const deploy = (links: ResolvedNavLink[]) => (isDefault === false ? links : []);
  return [
    ...deploy([
      {
        key: "apps",
        label: "Apps",
        href: routes.projects.detail(scope),
        icon: IconCubeOutline18,
        isActive: !page,
      },
    ]),
    {
      key: "keyspaces",
      label: "Keyspaces",
      href: routes.apis.list(scope),
      icon: IconNodesOutline18,
      isActive: page === "keyspaces",
    },
    {
      key: "ratelimits",
      label: "Ratelimits",
      href: routes.ratelimits.list(scope),
      icon: IconGaugeOutline18,
      isActive: page === "ratelimits",
    },
    {
      key: "authorization",
      label: "Authorization",
      href: routes.authorization.roles(scope),
      icon: IconShieldKeyOutline18,
      isActive: page === "authorization",
    },
    {
      key: "identities",
      label: "Identities",
      href: routes.identities.list(scope),
      icon: IconFingerprintOutline18,
      isActive: page === "identities",
    },
    ...deploy([
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
    ]),
    {
      key: "settings",
      label: "Project Settings",
      href: routes.projects.settings(scope),
      icon: IconGearOutline18,
      isActive: page === "settings",
    },
  ];
}

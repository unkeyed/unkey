import type { Workspace } from "@/lib/db";
import {
  Cube,
  Fingerprint,
  Gauge,
  Gear,
  Grid,
  InputSearch,
  Layers3,
  Nodes,
  ShieldKey,
  Sparkle3,
} from "@unkey/icons";
import { cn } from "../../../lib/utils";

export type NavItem = {
  disabled?: boolean;
  tooltip?: string;
  icon: React.ElementType | null;
  href: string;
  external?: boolean;
  label: string | React.ReactNode;
  active?: boolean;
  tag?: React.ReactNode;
  hidden?: boolean;
  items?: NavItem[];
  loadMoreAction?: boolean;
  showSubItems?: boolean;
};

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

export const createWorkspaceNavigation = (segments: string[], workspace: Workspace) => {
  const basePath = `/${workspace.slug}`;
  return [
    {
      icon: Nodes,
      href: `${basePath}/apis`,
      label: "APIs",
      active: segments.at(1) === "apis",
      showSubItems: false,
    },
    {
      icon: Cube,
      href: `${basePath}/projects`,
      label: "Projects",
      active: segments.at(1) === "projects",
      hidden: !workspace?.betaFeatures.deployments,
      tag: <Tag label="Beta" className="mr-2 group-hover:bg-gray-1" />,
    },
    {
      icon: Gauge,
      href: `${basePath}/ratelimits`,
      label: "Ratelimit",
      active: segments.at(1) === "ratelimits",
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
      icon: InputSearch,
      href: `${basePath}/audit`,
      label: "Audit Log",
      active: segments.at(1) === "audit",
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
      icon: Sparkle3,
      href: "/success",
      label: "Success",
      active: segments.at(1) === "success",
      tag: <Tag label="Internal" />,
      hidden: !workspace?.features.successPage,
    },
    {
      icon: Fingerprint,
      href: `${basePath}/identities`,
      label: "Identities",
      active: segments.at(1) === "identities",
      hidden: !workspace?.betaFeatures.identities,
    },
    {
      icon: Gear,
      href: `${basePath}/settings/general`,
      label: "Settings",
      active: segments.at(1) === "settings",
      items: [
        {
          icon: null,
          href: `${basePath}/settings/general`,
          label: "General",
          active: segments.some((s) => s === "general"),
        },
        {
          icon: null,
          href: `${basePath}/settings/team`,
          label: "Team",
          active: segments.some((s) => s === "team"),
        },
        {
          icon: null,
          href: `${basePath}/settings/root-keys`,
          label: "Root Keys",
          active: segments.some((s) => s === "root-keys"),
        },
        {
          icon: null,
          href: `${basePath}/settings/billing`,
          label: "Billing",
          active: segments.some((s) => s === "billing"),
        },
      ],
    },
  ].filter((n) => !n.hidden);
};

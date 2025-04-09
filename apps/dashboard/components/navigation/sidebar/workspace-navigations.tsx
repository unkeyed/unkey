import type { Workspace } from "@/lib/db";
import {
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
  label: string;
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

export const createWorkspaceNavigation = (
  workspace: Pick<Workspace, "features" | "betaFeatures">,
  segments: string[],
) => {
  return [
    {
      icon: Nodes,
      href: "/apis",
      label: "APIs",
      active: segments.at(0) === "apis",
      showSubItems: false,
    },
    {
      icon: Gauge,
      href: "/ratelimits",
      label: "Ratelimit",
      active: segments.at(0) === "ratelimits",
    },
    {
      icon: ShieldKey,
      label: "Authorization",
      href: "/authorization/roles",
      active: segments.some((s) => s === "authorization"),
      items: [
        {
          icon: null,
          label: "Roles",
          href: "/authorization/roles",
          active: segments.some((s) => s === "roles"),
        },
        {
          icon: null,
          label: "Permissions",
          href: "/authorization/permissions",
          active: segments.some((s) => s === "permissions"),
        },
      ],
    },

    {
      icon: InputSearch,
      href: "/audit",
      label: "Audit Log",
      active: segments.at(0) === "audit",
    },
    {
      icon: Grid,
      href: "/monitors/verifications",
      label: "Monitors",
      active: segments.at(0) === "verifications",
      hidden: !workspace.features.webhooks,
    },
    {
      icon: Layers3,
      href: "/logs",
      label: "Logs",
      active: segments.at(0) === "logs",
    },
    {
      icon: Sparkle3,
      href: "/success",
      label: "Success",
      active: segments.at(0) === "success",
      tag: <Tag label="Internal" />,
      hidden: !workspace.features.successPage,
    },
    {
      icon: Fingerprint,
      href: "/identities",
      label: "Identities",
      active: segments.at(0) === "identities",
      hidden: !workspace.betaFeatures.identities,
    },
    {
      icon: Gear,
      href: "/settings/general",
      label: "Settings",
      active: segments.at(0) === "settings",
      items: [
        {
          icon: null,
          href: "/settings/team",
          label: "Team",
          active: segments.some((s) => s === "team"),
        },
        {
          icon: null,
          href: "/settings/root-keys",
          label: "Root Keys",
          active: segments.some((s) => s === "root-keys"),
        },
        {
          icon: null,
          href: "/settings/billing",
          label: "Billing",
          active: segments.some((s) => s === "billing"),
        },
      ],
    },
  ].filter((n) => !n.hidden);
};

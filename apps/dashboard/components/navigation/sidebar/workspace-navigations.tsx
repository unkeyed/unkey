import type { Workspace } from "@/lib/db";
import {
  BookBookmark,
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

const DiscordIcon = () => (
  <svg
    className="w-4 h-4 fill-current" // Increased height to 6
    viewBox="0 -28.5 256 256"
    version="1.1"
    xmlns="http://www.w3.org/2000/svg"
    preserveAspectRatio="xMidYMid"
  >
    <g>
      <path
        d="M216.856339,16.5966031 C200.285002,8.84328665 182.566144,3.2084988 164.041564,0 C161.766523,4.11318106 159.108624,9.64549908 157.276099,14.0464379 C137.583995,11.0849896 118.072967,11.0849896 98.7430163,14.0464379 C96.9108417,9.64549908 94.1925838,4.11318106 91.8971895,0 C73.3526068,3.2084988 55.6133949,8.86399117 39.0420583,16.6376612 C5.61752293,67.146514 -3.4433191,116.400813 1.08711069,164.955721 C23.2560196,181.510915 44.7403634,191.567697 65.8621325,198.148576 C71.0772151,190.971126 75.7283628,183.341335 79.7352139,175.300261 C72.104019,172.400575 64.7949724,168.822202 57.8887866,164.667963 C59.7209612,163.310589 61.5131304,161.891452 63.2445898,160.431257 C105.36741,180.133187 151.134928,180.133187 192.754523,160.431257 C194.506336,161.891452 196.298154,163.310589 198.110326,164.667963 C191.183787,168.842556 183.854737,172.420929 176.223542,175.320965 C180.230393,183.341335 184.861538,190.991831 190.096624,198.16893 C211.238746,191.588051 232.743023,181.531619 254.911949,164.955721 C260.227747,108.668201 245.831087,59.8662432 216.856339,16.5966031 Z M85.4738752,135.09489 C72.8290281,135.09489 62.4592217,123.290155 62.4592217,108.914901 C62.4592217,94.5396472 72.607595,82.7145587 85.4738752,82.7145587 C98.3405064,82.7145587 108.709962,94.5189427 108.488529,108.914901 C108.508531,123.290155 98.3405064,135.09489 85.4738752,135.09489 Z M170.525237,135.09489 C157.88039,135.09489 147.510584,123.290155 147.510584,108.914901 C147.510584,94.5396472 157.658606,82.7145587 170.525237,82.7145587 C183.391518,82.7145587 193.761324,94.5189427 193.539891,108.914901 C193.539891,123.290155 183.391518,135.09489 170.525237,135.09489 Z"
        fillRule="nonzero"
      />
    </g>
  </svg>
);

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
        {
          icon: null,
          href: "/settings/vercel",
          label: "Vercel Integration",
          active: segments.some((s) => s === "vercel"),
        },
        {
          icon: null,
          href: "/settings/user",
          label: "User",
          active: segments.some((s) => s === "user"),
        },
      ],
    },
  ].filter((n) => !n.hidden);
};

export const resourcesNavigation: NavItem[] = [
  {
    icon: BookBookmark,
    href: "https://unkey.dev/docs",
    external: true,
    label: "Docs",
  },
  {
    icon: DiscordIcon,
    href: "https://www.unkey.com/discord",
    external: true,
    label: "Discord",
  },
];

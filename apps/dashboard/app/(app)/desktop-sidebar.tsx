"use client";
import { Feedback } from "@/components/dashboard/feedback-component";
import { Badge } from "@/components/ui/badge";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import type { Workspace } from "@/lib/db";
import { cn } from "@/lib/utils";
import {
  BookOpen,
  Cable,
  Crown,
  DatabaseZap,
  Fingerprint,
  Gauge,
  List,
  Loader2,
  type LucideIcon,
  MonitorDot,
  Settings2,
  ShieldCheck,
} from "lucide-react";
import Link from "next/link";
import { useSelectedLayoutSegments } from "next/navigation";
import { useRouter } from "next/navigation";
import type React from "react";
import { useTransition } from "react";
import { WorkspaceSwitcher } from "./team-switcher";
import { UserButton } from "./user-button";
type Props = {
  workspace: Workspace & {
    apis: {
      id: string;
      name: string;
    }[];
  };
  className?: string;
};

type NavItem = {
  disabled?: boolean;
  tooltip?: string;
  icon: LucideIcon | React.ElementType;
  href: string;
  external?: boolean;
  label: string;
  active?: boolean;
  tag?: React.ReactNode;
  hidden?: boolean;
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
      "bg-background border text-content-subtle rounded text-xs px-1 py-0.5  font-mono ",
      className,
    )}
  >
    {label}
  </div>
);

export const DesktopSidebar: React.FC<Props> = ({ workspace, className }) => {
  const segments = useSelectedLayoutSegments() ?? [];
  const workspaceNavigation: NavItem[] = [
    {
      icon: Cable,
      href: "/apis",
      label: "APIs",
      active: segments.at(0) === "apis",
    },
    {
      icon: Gauge,
      href: "/ratelimits",
      label: "Ratelimit",
      active: segments.at(0) === "ratelimits",
    },
    {
      icon: ShieldCheck,
      label: "Authorization",
      href: "/authorization/roles",
      active: segments.some((s) => s === "authorization"),
    },

    {
      icon: List,
      href: "/audit",
      label: "Audit Log",
      active: segments.at(0) === "audit",
    },
    {
      icon: MonitorDot,
      href: "/monitors/verifications",
      label: "Monitors",
      active: segments.at(0) === "verifications",
      hidden: !workspace.features.webhooks,
    },
    {
      icon: Crown,
      href: "/success",
      label: "Success",
      active: segments.at(0) === "success",
      tag: <Tag label="internal" />,
      hidden: !workspace.features.successPage,
    },
    {
      icon: DatabaseZap,
      href: "/semantic-cache",
      label: "Semantic Cache",
      active: segments.at(0) === "semantic-cache",
    },
    {
      icon: Fingerprint,
      href: "/identities",
      label: "Identities",
      active: segments.at(0) === "identities",
      hidden: !workspace.betaFeatures.identities,
    },
    {
      icon: Settings2,
      href: "/settings/general",
      label: "Settings",
      active: segments.at(0) === "settings",
    },
  ].filter((n) => !n.hidden);
  const resourcesNavigation: NavItem[] = [
    {
      icon: BookOpen,
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

  const firstOfNextMonth = new Date();
  firstOfNextMonth.setUTCMonth(firstOfNextMonth.getUTCMonth() + 1);
  firstOfNextMonth.setDate(1);
  return (
    <aside
      className={cn(
        "bg-background text-content/65 inset-y-0 w-64 px-5 z-10 h-full shrink-0 flex flex-col overflow-y-auto",
        className,
      )}
    >
      <div className="flex min-w-full mt-2 -mx-2">
        <WorkspaceSwitcher />
      </div>
      {workspace.planDowngradeRequest ? (
        <div className="flex justify-center w-full mt-2">
          <Tooltip>
            <TooltipTrigger>
              <Badge size="sm">Subscription ending</Badge>
            </TooltipTrigger>
            <TooltipContent>
              Your plan is schedueld to be downgraded to the {workspace.planDowngradeRequest} tier
              on {firstOfNextMonth.toDateString()}
            </TooltipContent>
          </Tooltip>
        </div>
      ) : null}
      <nav className="flex flex-col flex-1 flex-grow mt-6 pb-10">
        <ul className="flex flex-col flex-1 gap-y-6">
          <li className="flex flex-col gap-2">
            <h2 className="text-xs leading-6 uppercase">Workspace</h2>
            <ul className="-mx-2 space-y-1">
              {workspaceNavigation.map((item) => (
                <li key={item.label}>
                  <NavLink item={item} />
                </li>
              ))}
            </ul>
          </li>
          <li className="flex flex-col gap-2">
            <h2 className="text-xs leading-6 uppercase">Resources</h2>
            <ul className="-mx-2 space-y-1">
              {resourcesNavigation.map((item) => (
                <li key={item.label}>
                  <NavLink item={item} />
                </li>
              ))}
              <li>
                <Feedback />
              </li>
            </ul>
          </li>
        </ul>
      </nav>

      <div className="bg-[inherit] min-w-full [flex:0_0_56px] -mx-2 sticky bottom-0">
        <UserButton />

        {/* Fading indicator that there are more items to scroll */}
        <div className="pointer-events-none absolute bottom-full inset-x-0 h-10 bg-[inherit] [mask-image:linear-gradient(to_top,white,transparent)]" />
      </div>
    </aside>
  );
};

const NavLink: React.FC<{ item: NavItem }> = ({ item }) => {
  const [isPending, startTransition] = useTransition();
  const router = useRouter();
  const link = (
    <Link
      prefetch
      href={item.href}
      onClick={() => {
        if (!item.external) {
          startTransition(() => {
            router.push(item.href);
          });
        }
      }}
      target={item.external ? "_blank" : undefined}
      className={cn(
        "transition-all duration-150 group flex gap-x-2 rounded-md px-2 py-1 text-sm font-normal leading-6 items-center border border-transparent hover:bg-background-subtle hover:text-content justify-between",
        {
          "bg-background border-border text-content [box-shadow:0px_1px_3px_0px_rgba(0,0,0,0.03)]":
            item.active,
          "text-content-subtle pointer-events-none": item.disabled,
        },
      )}
    >
      <div className="flex items-center group gap-x-2">
        <span className="flex h-5 w-5 shrink-0 items-center justify-center text-[0.625rem]">
          {isPending ? (
            <Loader2 className="w-5 h-5 shrink-0 animate-spin" />
          ) : (
            <item.icon className="w-5 h-5 shrink-0 [stroke-width:1.25px]" aria-hidden="true" />
          )}
        </span>
        <p className="truncate whitespace-nowrap">{item.label}</p>
      </div>
      {item.tag}
    </Link>
  );

  if (item.tooltip) {
    return (
      <Tooltip>
        <TooltipTrigger className="w-full">
          {link}
          <TooltipContent>{item.tooltip}</TooltipContent>
        </TooltipTrigger>
      </Tooltip>
    );
  }
  return link;
};

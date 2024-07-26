"use client";
import { Badge } from "@/components/ui/badge";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import type { Workspace } from "@/lib/db";
import { cn } from "@/lib/utils";
import {
  Cable,
  Crown,
  DatabaseZap,
  ExternalLink,
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
  icon: LucideIcon;
  href: string;
  external?: boolean;
  label: string;
  active?: boolean;
  tag?: React.ReactNode;
  hidden?: boolean;
};

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
  const segments = useSelectedLayoutSegments();
  const workspaceNavigation: NavItem[] = [
    {
      icon: Cable,
      href: "/apis",
      label: "APIs",
      active: segments.length === 1 && segments.at(0) === "apis",
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
      icon: Settings2,
      href: "/settings/general",
      label: "Settings",
      active: segments.at(0) === "settings",
    },
  ].filter((n) => !n.hidden);
  const resourcesNavigation: NavItem[] = [
    {
      icon: ExternalLink,
      href: "https://unkey.dev/docs",
      external: true,
      label: "Docs",
    },
  ];

  const firstOfNextMonth = new Date();
  firstOfNextMonth.setUTCMonth(firstOfNextMonth.getUTCMonth() + 1);
  firstOfNextMonth.setDate(1);

  return (
    <aside
      className={cn(
        "bg-background text-content/65 inset-y-0 w-64 px-5 z-10 h-screen flex flex-col",
        className,
      )}
    >
      <div className="[flex:1]">
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
        <nav className="flex flex-col flex-1 flex-grow mt-6">
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
              </ul>
            </li>
          </ul>
        </nav>
      </div>

      <div className="min-w-full h-fit [flex:0_0_56px] -mx-2 mb-2">
        <UserButton />
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
        "transition-all duration-150 group flex gap-x-2 rounded-md px-2 py-1 text-sm font-normal leading-6 items-center hover:bg-gray-100 hover:dark:bg-gray-950 hover:text-content justify-between",
        {
          "bg-gray-100 dark:bg-gray-950 border border-border text-content font-medium": item.active,
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

"use client";
import { Badge } from "@/components/ui/badge";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import type { Workspace } from "@/lib/db";
import { cn } from "@/lib/utils";
import {
  Activity,
  BookOpen,
  Code,
  Crown,
  DatabaseZap,
  GlobeLock,
  Loader2,
  type LucideIcon,
  MonitorDot,
  ReceiptText,
  Settings,
  ShieldHalf,
  Webhook,
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
  const navigation: NavItem[] = [
    {
      icon: Code,
      href: "/apis",
      label: "APIs",
      active: segments.length === 1 && segments.at(0) === "apis",
    },

    {
      icon: BookOpen,
      href: "https://unkey.dev/docs",
      external: true,
      label: "Docs",
    },
    {
      icon: GlobeLock,
      href: "/ratelimits",
      label: "Ratelimit",
      active: segments.at(0) === "ratelimits",
    },
    {
      icon: ShieldHalf,
      label: "Authorization",
      href: "/authorization/roles",
      active: segments.some((s) => s === "authorization"),
    },

    {
      icon: Activity,
      href: "/audit",
      label: "Audit Log",
      active: segments.at(0) === "audit",
    },
    {
      icon: Settings,
      href: "/settings/general",
      label: "Settings",
      active: segments.at(0) === "settings",
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
  ].filter((n) => !n.hidden);

  const firstOfNextMonth = new Date();
  firstOfNextMonth.setUTCMonth(firstOfNextMonth.getUTCMonth() + 1);
  firstOfNextMonth.setDate(1);

  return (
    <aside className={cn("fixed inset-y-0 w-64 px-6 z-10", className)}>
      <div className="flex min-w-full mt-4 -mx-2">
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
      <nav className="flex flex-col flex-1 flex-grow mt-4">
        <ul className="flex flex-col flex-1 gap-y-7">
          <li>
            <h2 className="text-xs font-semibold leading-6 text-content">General</h2>
            <ul className="mt-2 -mx-2 space-y-1">
              {navigation.map((item) => (
                <li key={item.label}>
                  <NavLink item={item} />
                </li>
              ))}
            </ul>
          </li>
        </ul>
      </nav>

      <UserButton />
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
        "group flex gap-x-2 rounded-md px-2 py-1 text-sm  font-medium leading-6 items-center hover:bg-gray-200 dark:hover:bg-gray-800 justify-between",
        {
          "bg-gray-200 dark:bg-gray-800": item.active,
          "text-content-subtle pointer-events-none": item.disabled,
        },
      )}
    >
      <div className="flex group gap-x-2">
        <span className="text-content-subtle border-border group-hover:shadow  flex h-6 w-6 shrink-0 items-center justify-center rounded-lg border text-[0.625rem] font-medium bg-white">
          {isPending ? (
            <Loader2 className="w-4 h-4 shrink-0 animate-spin" />
          ) : (
            <item.icon className="w-4 h-4 shrink-0" aria-hidden="true" />
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

"use client";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import type { Workspace } from "@/lib/db";
import { cn } from "@/lib/utils";
import { BookOpen, Code, LucideIcon, Settings } from "lucide-react";
import Link from "next/link";
import { useSelectedLayoutSegments } from "next/navigation";
import React from "react";
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
  icon: LucideIcon;
  href: string;
  external?: boolean;
  label: string;
  active?: boolean;
};

export const DesktopSidebar: React.FC<Props> = ({ workspace, className }) => {
  const segments = useSelectedLayoutSegments();
  const navigation: NavItem[] = [
    {
      icon: Code,
      href: "/app/apis",
      label: "APIs",
      active: segments.length === 1 && segments.at(0) === "apis",
    },
    {
      icon: Settings,
      href: "/app/settings/general",
      label: "Settings",
      active: segments.at(0) === "settings",
    },
    {
      icon: BookOpen,
      href: "https://unkey.dev/docs",
      external: true,
      label: "Docs",
    },
  ];

  return (
    <aside className={cn("fixed inset-y-0 w-64 px-6 z-10", className)}>
      <div className="flex -mx-2 mt-4 min-w-full">
        <WorkspaceSwitcher />
      </div>
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
          <li>
            <h2 className="text-xs font-semibold leading-6 text-content">Your APIs</h2>
            {/* max-h-64 in combination with the h-8 on the <TooltipTrigger> will fit 8 apis nicely */}
            <ScrollArea className="mt-2 max-h-64 -mx-2 space-y-1 overflow-auto">
              {workspace.apis.map((api) => (
                <Tooltip key={api.id}>
                  <TooltipTrigger className="w-full h-8 overflow-hidden text-ellipsis">
                    <NavLink
                      item={{
                        icon: Code,
                        href: `/app/apis/${api.id}`,
                        label: api.name,
                        active: segments.includes(api.id),
                      }}
                    />
                  </TooltipTrigger>
                  <TooltipContent>{api.name}</TooltipContent>
                </Tooltip>
              ))}
            </ScrollArea>
          </li>
        </ul>
      </nav>

      <UserButton />
    </aside>
  );
};

const NavLink: React.FC<{ item: NavItem }> = ({ item }) => {
  return (
    <Link
      href={item.href}
      target={item.external ? "_blank" : undefined}
      className={cn(
        "group flex gap-x-2 rounded-md px-2 py-1 text-sm  font-medium leading-6 items-center hover:bg-gray-200 dark:hover:bg-gray-800",
        {
          "bg-gray-200 dark:bg-gray-800": item.active,
        },
      )}
    >
      <span className="text-content-subtle border-border group-hover:shadow  flex h-6 w-6 shrink-0 items-center justify-center rounded-lg border text-[0.625rem] font-medium bg-white">
        <item.icon className="w-4 h-4 shrink-0" aria-hidden="true" />
      </span>
      <p className="whitespace-nowrap truncate">{item.label}</p>
    </Link>
  );
};

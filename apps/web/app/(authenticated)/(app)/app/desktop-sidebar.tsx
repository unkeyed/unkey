"use client";
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
      active: segments.at(0) === "apis",
    },
    {
      icon: Settings,
      href: "/app/settings/general",
      label: "Settings",
      active: segments.at(0) === "settings",
    },
    {
      icon: BookOpen,
      href: "https://docs.unkey.dev",
      external: true,
      label: "Docs",
    },
  ];

  return (
    <aside className={cn("fixed  h-screen  inset-y-0 flex w-64 flex-col px-6 gap-y-5", className)}>
      <div className="flex items-center h-16 mt-4 shrink-0">
        <WorkspaceSwitcher />
      </div>
      <nav className="flex flex-col flex-1 flex-grow">
        <ul role="list" className="flex flex-col flex-1 gap-y-7">
          <li>
            <h3 className="text-xs font-semibold leading-6 text-content">General</h3>
            <ul role="list" className="mt-2 -mx-2 space-y-1">
              {navigation.map((item) => (
                <li key={item.label}>
                  <NavLink item={item} />
                </li>
              ))}
            </ul>
          </li>
          <li>
            <h3 className="text-xs font-semibold leading-6 text-content">Your APIs</h3>
            <ul role="list" className="mt-2 -mx-2 space-y-1">
              {workspace.apis.map((api) => (
                <li key={api.id}>
                  <NavLink
                    item={{
                      icon: Code,
                      href: `/app/apis/${api.id}`,
                      label: api.name,
                      active: segments.includes(api.id),
                    }}
                  />
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
  return (
    <Link
      href={item.href}
      target={item.external ? "_blank" : undefined}
      className={cn(
        "group flex gap-x-2 rounded-md px-2 py-1 text-sm  font-medium leading-6 items-center hover:bg-gray-200 dark:hover:bg-gray-800 ",
        {
          "bg-gray-200 dark:bg-gray-800": item.active,
        },
      )}
    >
      <span className="text-content-subtle border-border group-hover:shadow  flex h-6 w-6 shrink-0 items-center justify-center rounded-lg border text-[0.625rem] font-medium bg-white">
        <item.icon className="w-4 h-4 shrink-0" aria-hidden="true" />
      </span>
      {item.label}
    </Link>
  );
};

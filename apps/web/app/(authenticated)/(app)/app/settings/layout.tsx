"use client";

import Link from "next/link";
import * as React from "react";

import { Separator } from "@/components/ui/separator";
import { cn } from "@/lib/utils";
import { useSelectedLayoutSegment } from "next/navigation";

export default function SettingsLayout({ children }: { children: React.ReactNode }) {
  const segment = useSelectedLayoutSegment();
  const navigation: { label: string; href: string; active?: boolean }[] = [
    {
      label: "General",
      href: "/app/settings/general",
      active: segment === "general",
    },
    {
      label: "Team",
      href: "/app/settings/team",
      active: segment === "team",
    },
    {
      label: "Root Keys",
      href: "/app/settings/root-keys",
      active: segment === "root-keys",
    },
    {
      label: "Billing",
      href: "/app/stripe",
    },
    {
      label: "Usage",
      href: "/app/settings/usage",
      active: segment === "usage",
    },
    {
      label: "User",
      href: "/app/settings/user",
      active: segment === "user",
    },
  ];

  return (
    <div>
      <div className="space-y-1 ">
        <h2 className="text-2xl font-semibold tracking-tight">Settings</h2>
        <p className="text-sm text-gray-500 dark:text-gray-400">Manage your workspace settings.</p>
      </div>
      <nav className="sticky top-0 bg-background">
        <div className="flex items-center w-full gap-4 mt-8">
          {navigation.map(({ label, href, active }) => (
            <li
              className={cn(" list-none border-b-2 border-transparent p-2 ", {
                "border-primary ": active,
              })}
            >
              <Link
                href={href}
                className={cn(
                  "text-sm font-medium py-2 px-3 -mx-3 text-content-subtle  hover:bg-primary rounded-md hover:text-primary-foreground",
                  {
                    "text-primary": active,
                  },
                )}
              >
                {label}
              </Link>
            </li>
          ))}
        </div>
        <Separator className="" />
      </nav>

      <main className="mt-8">{children}</main>
    </div>
  );
}

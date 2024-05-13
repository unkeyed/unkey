import type * as React from "react";

import { Navbar } from "@/components/dashboard/navbar";

export const dynamic = "force-dynamic";
export const runtime = "edge";

export default function SettingsLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const navigation = [
    {
      label: "General",
      href: "/settings/general",
      segment: "general",
    },
    {
      label: "Team",
      href: "/settings/team",
      segment: "team",
    },
    {
      label: "Root Keys",
      href: "/settings/root-keys",
      segment: "root-keys",
    },
    {
      label: "Billing",
      href: "/settings/billing",
      segment: "billing",
    },
    {
      label: "Vercel Integration",
      href: "/settings/vercel",
      segment: "vercel",
    },
    // {
    //   label: "Webhooks",
    //   href: "/settings/webhooks",
    //   segment: "webhooks",
    // },
    {
      label: "User",
      href: "/settings/user",
      segment: "user",
    },
  ];

  return (
    <div>
      <div className="space-y-1 ">
        <h1 className="text-2xl font-semibold tracking-tight">Settings</h1>
        <p className="text-sm text-gray-500 dark:text-gray-400">Manage your workspace settings.</p>
      </div>

      <Navbar navigation={navigation} className="mt-8" />

      <main className="mt-8 mb-20">{children}</main>
    </div>
  );
}

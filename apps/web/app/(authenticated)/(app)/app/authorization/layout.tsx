import * as React from "react";

import { Navbar } from "@/components/dashboard/navbar";
import { PageHeader } from "@/components/dashboard/page-header";

export const dynamic = "force-dynamic";
export const runtime = "edge";

export default function SettingsLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const navigation = [
    {
      label: "Roles",
      href: "/app/authorization/roles",
      segment: "roles",
    },
    {
      label: "Permissions",
      href: "/app/authorization/permissions",
      segment: "permissions",
    },
  ];

  return (
    <div>
      <PageHeader title="Authorization" description="Manage your roles and permissions" />

      <Navbar navigation={navigation} className="mt-8" />

      <main className="mt-8 mb-20">{children}</main>
    </div>
  );
}

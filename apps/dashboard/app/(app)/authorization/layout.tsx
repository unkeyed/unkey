import type * as React from "react";

import { Navbar } from "@/components/dashboard/navbar";
import { getTenantId } from "@/lib/auth";
import { db } from "@/lib/db";
import { redirect } from "next/navigation";

export const dynamic = "force-dynamic";
export const runtime = "edge";

export default async function AuthorizationLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const tenantId = getTenantId();
  const workspace = await db.query.workspaces.findFirst({
    where: (table, { eq }) => eq(table.tenantId, tenantId),
  });
  if (!workspace) {
    return redirect("/auth/sign-in");
  }

  const navigation = [
    {
      label: "Roles",
      href: "/authorization/roles",
      segment: "roles",
    },
    {
      label: "Permissions",
      href: "/authorization/permissions",
      segment: "permissions",
    },
  ];

  return (
    <div>
      <Navbar navigation={navigation} className="mt-8" />

      <main className="mt-8 mb-20 overflow-x-auto">{children}</main>
    </div>
  );
}

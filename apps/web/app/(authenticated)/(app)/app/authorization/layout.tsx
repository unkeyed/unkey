import * as React from "react";

import { Banner } from "@/components/banner";
import { Navbar } from "@/components/dashboard/navbar";
import { PageHeader } from "@/components/dashboard/page-header";
import { OptIn } from "@/components/opt-in";
import { getTenantId } from "@/lib/auth";
import { db } from "@/lib/db";
import Link from "next/link";
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
  if (!workspace.betaFeatures.rbac) {
    return (
      <OptIn
        title="Authorization is still in Alpha"
        description="Do you want to enable this feature for your workspace?"
        feature="rbac"
      />
    );
  }

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
      <Banner>
        <p className="text-xs text-center">
          RBAC is in alpha, feel free to play around, but this is not production ready yet. Please{" "}
          <Link href="mailto:support@unkey.dev" className="underline">
            reach out
          </Link>{" "}
          for more information.
        </p>
      </Banner>
      <PageHeader title="Authorization" description="Manage your roles and permissions" />

      <Navbar navigation={navigation} className="mt-8" />

      <main className="mt-8 mb-20">{children}</main>
    </div>
  );
}

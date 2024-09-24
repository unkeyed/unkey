import type * as React from "react";

import { Banner } from "@/components/banner";
import { Navbar } from "@/components/dashboard/navbar";
import { PageHeader } from "@/components/dashboard/page-header";
import { OptIn } from "@/components/opt-in";
import { serverAuth } from "@/lib/auth/server";
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
  const tenantId = await serverAuth.getTenantId();
  const workspace = await db.query.workspaces.findFirst({
    where: (table, { eq }) => eq(table.tenantId, tenantId),
  });
  if (!workspace) {
    return redirect("/auth/sign-in");
  }

  if (!workspace.betaFeatures.identities) {
    children = (
      <OptIn title="Identities" description="Identities are in beta" feature="identities" />
    );
  }
  return (
    <div>
      <main className="mt-8 mb-20 overflow-x-auto">{children}</main>
    </div>
  );
}

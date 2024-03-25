import { Navbar } from "@/components/dashboard/navbar";
import { PageHeader } from "@/components/dashboard/page-header";
import { Badge } from "@/components/ui/badge";
import { getTenantId } from "@/lib/auth";
import { db } from "@/lib/db";
import { redirect } from "next/navigation";
import * as React from "react";

export const dynamic = "force-dynamic";
export const runtime = "edge";

export default async function AnalyticsLayout({
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

  return (
    <div>
      <PageHeader title="Analytics" description="View your analytics data" actions={[]} />
      <main className="relative mt-8 mb-20 ">{children}</main>
    </div>
  );
}

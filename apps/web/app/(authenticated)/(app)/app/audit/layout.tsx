import { OptIn } from "@/components/opt-in";
import { getTenantId } from "@/lib/auth";
import { db } from "@/lib/db";
import { redirect } from "next/navigation";
import * as React from "react";

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
  if (!workspace.betaFeatures.auditLogRetentionDays) {
    return (
      <OptIn
        title="Audit Logs are in beta"
        description="Do you want to enable this feature for your workspace?"
        feature="auditLogRetentionDays"
      />
    );
  }

  return <>{children}</>;
}

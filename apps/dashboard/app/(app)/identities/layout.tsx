import type * as React from "react";

import { OptIn } from "@/components/opt-in";
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

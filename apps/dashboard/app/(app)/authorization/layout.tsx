import type * as React from "react";

import { getAuth } from "@/lib/auth";
import { db } from "@/lib/db";
import { redirect } from "next/navigation";

export const dynamic = "force-dynamic";

export default async function AuthorizationLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const { orgId } = await getAuth();
  const workspace = await db.query.workspaces.findFirst({
    where: (table, { eq }) => eq(table.orgId, orgId),
  });
  if (!workspace) {
    return redirect("/auth/sign-in");
  }

  return children;
}

import { Banner } from "@/components/banner";
import { OptIn } from "@/components/opt-in";
import { getTenantId } from "@/lib/auth";
import { db } from "@/lib/db";
import Link from "next/link";
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
  if (!workspace.betaFeatures.ratelimit) {
    return (
      <OptIn
        title="Ratelimiting is in beta"
        description="Do you want to enable ratelimiting for your workspace?"
        feature="ratelimit"
      />
    );
  }

  return (
    <>
      <Banner persistChoice="ratelimit_ui">
        <p className="text-xs text-center">
          The Ratelimiting UI is still in development. Please{" "}
          <Link href="mailto:support@unkey.dev" className="underline">
            reach out
          </Link>{" "}
          for feedback.
        </p>
      </Banner>
      {children}
    </>
  );
}

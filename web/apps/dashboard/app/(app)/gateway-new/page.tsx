/**
 * This route creates a shortcut for onboarding
 *
 * We need this for compliance of our gateway.new domain.
 * 1. A user will enter "gateway.new" in the browser
 * 2. Vercel will detect the host and rewrite the request to this page
 * 3. A workspace is upserted
 * 4. The user is redirected to create their API
 */
import { randomInt } from "node:crypto";
import { getAuth } from "@/lib/auth";
import { db, schema } from "@/lib/db";
import { freeTierLimits } from "@/lib/limits";
import { dns1035, newId } from "@unkey/id";
import { redirect } from "next/navigation";

export const dynamic = "force-dynamic";

export default async function Page() {
  const { orgId } = await getAuth();

  const ws = await db.query.workspaces.findFirst({
    where: (table, { eq, isNull, and }) => and(eq(table.orgId, orgId), isNull(table.deletedAtM)),
  });

  if (!ws) {
    const id = newId("workspace");
    await db.transaction(async (tx) => {
      await tx.insert(schema.workspaces).values({
        id,
        name: "Personal Workspace",
        slug: `personal-workspace-${randomInt(100000)}`,
        orgId,
        betaFeatures: {},
        k8sNamespace: dns1035(12),
      });
      await tx.insert(schema.limits).values({
        workspaceId: id,
        ...freeTierLimits,
      });
      await tx.insert(schema.workspaceBilling).values({
        workspaceId: id,
        tier: "Free",
      });
    });
  }

  return redirect("/apis?new=true");
}

/**
 * This route creates a shortcut for onboarding
 *
 * We need this for compliance of our gateway.new domain.
 * 1. A user will enter "gateway.new" in the browser
 * 2. Vercel will detect the host and rewrite the request to this page
 * 3. A workspace is upserted
 * 4. The user is redirected to create their API
 */

import { getAuthOrRedirect } from "@/lib/auth";
import { db, schema } from "@/lib/db";
import { freeTierQuotas } from "@/lib/quotas";
import { newId } from "@unkey/id";
import { redirect } from "next/navigation";

export const dynamic = "force-dynamic";

export default async function Page() {
  const { orgId } = await getAuthOrRedirect();
  if (!orgId) {
    redirect("/new");
  }
  const ws = await db.query.workspaces.findFirst({
    where: (table, { eq, isNull, and }) => and(eq(table.orgId, orgId), isNull(table.deletedAtM)),
  });

  if (!ws) {
    const id = newId("workspace");
    await db.insert(schema.workspaces).values({
      id,
      name: "Personal Workspace",
      orgId,
      betaFeatures: {},
      features: {},
    });

    await db.insert(schema.quotas).values({
      workspaceId: id,
      ...freeTierQuotas,
    });
  }

  return redirect("/apis?new=true");
}

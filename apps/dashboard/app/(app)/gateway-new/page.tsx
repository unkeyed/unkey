/**
 * This route creates a shortcut for onboarding
 *
 * We need this for compliance of our gateway.new domain.
 * 1. A user will enter "gateway.new" in the browser
 * 2. Vercel will detect the host and rewrite the request to this page
 * 3. A workspace is upserted
 * 4. The user is redirected to create their API
 */

import { getOrgId } from "@/lib/auth";
import { db, schema } from "@/lib/db";
import { newId } from "@unkey/id";
import { redirect } from "next/navigation";

export const dynamic = "force-dynamic";

export default async function Page() {
  const orgId = await getOrgId();

  const ws = await db.query.workspaces.findFirst({
    where: (table, { eq, isNull, and }) => and(eq(table.orgId, orgId), isNull(table.deletedAtM)),
  });

  if (!ws) {
    await db.insert(schema.workspaces).values({
      id: newId("workspace"),
      name: "Personal Workspace",
      orgId,
      // dumb hack to keep the unique property but also clearly mark it as a workos identifier
      clerkTenantId: `workos_${orgId}`,
      betaFeatures: {},
      features: {},
    });
  }

  return redirect("/apis?new=true");
}

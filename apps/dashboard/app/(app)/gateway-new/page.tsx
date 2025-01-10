/**
 * This route creates a shortcut for onboarding
 *
 * We need this for compliance of our gateway.new domain.
 * 1. A user will enter "gateway.new" in the browser
 * 2. Vercel will detect the host and rewrite the request to this page
 * 3. A workspace is upserted
 * 4. The user is redirected to create their API
 */

import { getTenantId } from "@/lib/auth";
import { db, schema } from "@/lib/db";
import { newId } from "@unkey/id";
import { redirect } from "next/navigation";

export default async function Page() {
  const tenantId = await getTenantId();

  const ws = await db.query.workspaces.findFirst({
    where: (table, { eq, isNull, and }) =>
      and(eq(table.tenantId, tenantId), isNull(table.deletedAtM)),
  });

  if (!ws) {
    await db.insert(schema.workspaces).values({
      id: newId("workspace"),
      name: "Personal Workspace",
      tenantId,
      betaFeatures: {},
      features: {},
    });
  }

  return redirect("/apis?new=true");
}

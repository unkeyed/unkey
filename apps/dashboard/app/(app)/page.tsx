import { getAuthOrRedirect } from "@/lib/auth";
import { db } from "@/lib/db";
import { redirect } from "next/navigation";

export const dynamic = "force-dynamic";

export default async function TenantOverviewPage() {
  const { orgId } = await getAuthOrRedirect();

  if (!orgId) {
    redirect("/new");
  }
  const workspace = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) => and(eq(table.orgId, orgId), isNull(table.deletedAtM)),
  });
  if (!workspace) {
    redirect("/new");
  }
  redirect("/apis");
}

import { getAuth } from "@/lib/auth";
import { db } from "@/lib/db";
import { redirect } from "next/navigation";

export const dynamic = "force-dynamic";

export default async function TenantOverviewPage() {
  const { orgId } = await getAuth();

  if (!orgId) {
    return redirect("/sign-in");
  }

  const workspace = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) => and(eq(table.orgId, orgId), isNull(table.deletedAtM)),
  });

  if (!workspace) {
    return redirect("/new");
  }

  return redirect(`/${workspace.slug}/apis`);
}

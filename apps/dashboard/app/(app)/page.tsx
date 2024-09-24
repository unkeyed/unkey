import { serverAuth } from "@/lib/auth/server";
import { db } from "@/lib/db";
import { redirect } from "next/navigation";

export default async function TenantOverviewPage() {
  const tenantId = await serverAuth.getTenantId();
  const workspace = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) =>
      and(eq(table.tenantId, tenantId), isNull(table.deletedAt)),
  });
  if (!workspace) {
    return redirect("/new");
  }
  return redirect("/apis");
}

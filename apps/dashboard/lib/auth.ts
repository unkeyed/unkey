import { auth } from "@/lib/auth/server";
import { redirect } from "next/navigation";
import { db } from "./db";

/**
 * Return the tenant id or a 404 not found page.
 *
 * The auth check should already be done at a higher level, and we're just returning 404 to make typescript happy.
 */
export async function getTenantId(): Promise<string> {
  const user = await auth.getCurrentUser();
  if (!user) {
    console.log("get tenant id: no user")
    return redirect("/auth/sign-in");
  }

  const { orgId } = user;
  if (!orgId) {
    console.log("get tenant id: no orgId")
    return redirect("/new");
  }
  
  return orgId;
}

// client friendly way to get workspace
export async function getWorkspace(tenantId: string): Promise<any> {
  if (!tenantId) {
    throw new Error("TenantId/orgId is required to look up workspace");
  }
  const workspace = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) =>
      and(eq(table.tenantId, tenantId), isNull(table.deletedAt)),
  });
  return workspace;
}

import { auth } from "@/lib/auth/server";
import { db } from "@/lib/db";

export async function getAvailableWorkspaces(userId: string, orgId?: string) {
  const orgIds = await auth.listActiveOrganizationIds(userId, orgId);

  if (orgIds.length === 0) {
    return [];
  }

  return db.query.workspaces.findMany({
    columns: { orgId: true, name: true },
    where: (table, { and, inArray, isNull }) =>
      and(inArray(table.orgId, orgIds), isNull(table.deletedAtM)),
  });
}

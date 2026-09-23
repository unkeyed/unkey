import { auth } from "@/lib/auth/server";
import { db } from "@/lib/db";

export async function getAvailableWorkspaces(userId: string, orgId?: string) {
  const memberships = await auth.listMemberships(userId, orgId);
  const orgIds = memberships.data
    .filter((membership) => membership.status === "active")
    .map((membership) => membership.organization.id);

  if (orgIds.length === 0) {
    return [];
  }

  return db.query.workspaces.findMany({
    columns: { orgId: true, name: true },
    where: (table, { and, inArray, isNull }) =>
      and(inArray(table.orgId, orgIds), isNull(table.deletedAtM)),
  });
}

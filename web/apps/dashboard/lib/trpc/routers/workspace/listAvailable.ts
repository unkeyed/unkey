import { auth } from "@/lib/auth/server";
import { db } from "@/lib/db";
import { protectedProcedure } from "../../trpc";

export async function getAvailableWorkspaces(userId: string, orgId?: string) {
  const memberships = await auth.listMemberships(userId);
  const orgIds = memberships.data
    .filter(
      (membership) =>
        membership.status === "active" &&
        (orgId === undefined || membership.organization.id === orgId),
    )
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

export const listAvailable = protectedProcedure.query(({ ctx }) =>
  getAvailableWorkspaces(ctx.user.id),
);

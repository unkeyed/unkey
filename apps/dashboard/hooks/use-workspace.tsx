import { getAuth } from "@/lib/auth";
import { db } from "@/lib/db";

export const useWorkspace = async () => {
  const { orgId } = await getAuth();

  if (!orgId) {
    return null;
  }

  const workspace = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) => and(eq(table.orgId, orgId), isNull(table.deletedAtM)),
    with: {
      apis: {
        where: (table, { isNull }) => isNull(table.deletedAtM),
      },
      quotas: true,
    },
  });

  return workspace ?? null;
};

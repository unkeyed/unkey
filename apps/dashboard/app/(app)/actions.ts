// lib/workspace.ts (or wherever you keep data fetching logic)
import { cache } from "react";
import "server-only"; // Ensure this only runs on the server
import { type Quotas, type Workspace, db } from "@/lib/db";
import { redirect } from "next/navigation";

export type WorkspaceWithQuota = (Workspace & { quotas: Quotas | null }) | null;

// Define the function that fetches the workspace
const getWorkspaceByOrgIdUncached = async (orgId: string | null): Promise<WorkspaceWithQuota> => {
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
      ratelimitNamespaces: {
        where: (table, { isNull }) => isNull(table.deletedAtM),
        columns: {
          id: true,
          name: true,
        },
      },
    },
  });

  if (!workspace) {
    return null;
  }
  return { ...workspace, quotas: workspace.quotas };
};

export const getWorkspaceByOrgId = cache(getWorkspaceByOrgIdUncached);

export const getRequiredWorkspaceByOrgId = cache(
  async (orgId: string | null): Promise<WorkspaceWithQuota> => {
    const workspace = await getWorkspaceByOrgIdUncached(orgId);
    if (!workspace) {
      redirect("/new");
    }
    return workspace;
  },
);

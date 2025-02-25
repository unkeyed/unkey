import { db } from "@/lib/db";
import { redirect } from "next/navigation";

export const getWorkspace = async (tenantId: string) => {
  try {
    const workspace = await db.query.workspaces.findFirst({
      where: (table, { eq, and, isNull }) =>
        and(eq(table.tenantId, tenantId), isNull(table.deletedAtM)),
      with: {
        auditLogBuckets: {
          columns: {
            id: true,
            name: true,
          },
          orderBy: (table, { asc }) => asc(table.createdAt),
        },
      },
    });

    if (!workspace) {
      return redirect("/auth/signin");
    }
    return workspace;
  } catch (error) {
    console.error(
      `Failed to fetch workspace for tenant ID ${tenantId}: ${
        error instanceof Error ? error.message : "Unknown error"
      }`
    );
    throw error;
  }
};

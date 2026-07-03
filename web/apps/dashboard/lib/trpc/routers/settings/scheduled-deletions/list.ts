import { db } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";

// ScheduledDeletion is the unified row shape returned to the dashboard.
// Resource-type-specific fields fold into `name` so the UI renders one
// table without per-type branches.
export type ScheduledDeletion = {
  resourceType: "project";
  resourceId: string;
  name: string;
  deletePermanentlyAt: number;
};

// listScheduledDeletions reads the deletions table (the source of truth
// for which resources are currently in a grace window) and joins each
// row's name from the right resource table. Adding a new resource type
// is a new branch in the name-lookup phase below; the deletions query
// itself never changes.
export const listScheduledDeletions = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .query(async ({ ctx }): Promise<ScheduledDeletion[]> => {
    const workspaceId = ctx.workspace.id;

    const deletionRows = await db.query.deletions.findMany({
      where: (table, { eq }) => eq(table.workspaceId, workspaceId),
      orderBy: (table, { asc }) => [asc(table.deletePermanentlyAt)],
    });

    if (deletionRows.length === 0) {
      return [];
    }

    const projectIds = deletionRows
      .filter((r) => r.resourceType === "project")
      .map((r) => r.resourceId);

    if (projectIds.length === 0) {
      return [];
    }

    const projectRows = await db.query.projects.findMany({
      where: (table, { inArray }) => inArray(table.id, projectIds),
      columns: { id: true, name: true },
    });

    const projectNames = new Map(projectRows.map((p) => [p.id, p.name]));

    return deletionRows.flatMap<ScheduledDeletion>((row) => {
      if (row.resourceType !== "project") {
        // Unknown resource_type (a future backend writing a type the
        // dashboard hasn't learned yet). Skip rather than crash.
        return [];
      }
      const name = projectNames.get(row.resourceId);
      // Drop the row if the underlying resource row is missing — this
      // can happen briefly during the hard-delete cascade (resource
      // gone, deletions row not yet removed) and we'd rather hide the
      // entry than render an empty name.
      return name
        ? [
            {
              resourceType: "project" as const,
              resourceId: row.resourceId,
              name,
              deletePermanentlyAt: row.deletePermanentlyAt,
            },
          ]
        : [];
    });
  });

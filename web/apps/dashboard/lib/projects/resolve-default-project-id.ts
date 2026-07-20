import type { Transaction } from "@unkey/db";

export async function resolveDefaultProjectId(
  tx: Transaction,
  workspaceId: string,
): Promise<string> {
  const project = await tx.query.projects.findFirst({
    columns: { id: true, slug: true },
    where: (table, { and, eq }) =>
      and(eq(table.workspaceId, workspaceId), eq(table.slug, "default")),
  });

  return project?.slug === "default" ? project.id : "";
}

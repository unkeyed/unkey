import { isDuplicateKeyError } from "@/lib/utils/db-errors";
import { type Transaction, and, eq, schema, sql } from "@unkey/db";
import { newId } from "@unkey/id";

export async function ensureDefaultProjectId(
  tx: Transaction,
  workspaceId: string,
): Promise<string> {
  const project = await tx.query.projects.findFirst({
    where: and(
      eq(schema.projects.workspaceId, workspaceId),
      sql`BINARY ${schema.projects.slug} = 'default'`,
    ),
    columns: { id: true },
  });

  if (project) {
    return project.id;
  }

  const projectId = newId("project");
  try {
    await tx.insert(schema.projects).values({
      id: projectId,
      workspaceId,
      name: "Default",
      slug: "default",
      deleteProtection: true,
      createdAt: Date.now(),
    });
    return projectId;
  } catch (error) {
    if (!isDuplicateKeyError(error)) {
      throw error;
    }
  }

  const [concurrentProject] = await tx
    .select({ id: schema.projects.id })
    .from(schema.projects)
    .where(
      and(
        eq(schema.projects.workspaceId, workspaceId),
        sql`BINARY ${schema.projects.slug} = 'default'`,
      ),
    )
    .limit(1)
    .for("update");

  if (!concurrentProject) {
    throw new Error(`Unable to ensure default project for workspace ${workspaceId}`);
  }

  return concurrentProject.id;
}

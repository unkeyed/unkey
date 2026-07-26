import {
	type Database,
	type Transaction,
	and,
	eq,
	schema,
	sql,
} from "@unkey/db";

export async function resolveDefaultProjectId(
	database: Database | Transaction,
	workspaceId: string,
): Promise<string> {
	const project = await database.query.projects.findFirst({
		where: and(
			eq(schema.projects.workspaceId, workspaceId),
			sql`BINARY ${schema.projects.slug} = 'default'`,
		),
		columns: { id: true },
	});

	if (!project) {
		throw new Error(`Default project not found for workspace ${workspaceId}`);
	}

	return project.id;
}

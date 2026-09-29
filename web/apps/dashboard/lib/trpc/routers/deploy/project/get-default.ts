import { and, db, eq, sql } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { projects } from "@unkey/db/src/schema";

export const getDefaultProject = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .query(async ({ ctx }) => {
    const [project] = await db
      .select({
        id: projects.id,
        name: projects.name,
        slug: projects.slug,
        createdAt: projects.createdAt,
      })
      .from(projects)
      .where(
        and(eq(projects.workspaceId, ctx.workspace.id), sql`BINARY ${projects.slug} = 'default'`),
      )
      .limit(1);
    return project ?? null;
  });

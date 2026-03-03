import { count, db, eq } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { projects } from "@unkey/db/src/schema";

export const creationContext = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .query(async ({ ctx }) => {
    const [projectCountResult, installation] = await Promise.all([
      db
        .select({ count: count() })
        .from(projects)
        .where(eq(projects.workspaceId, ctx.workspace.id)),
      db.query.githubAppInstallations.findFirst({
        where: (table, { eq: eqFn }) => eqFn(table.workspaceId, ctx.workspace.id),
        columns: { pk: true },
      }),
    ]);

    return {
      isFirstProject: (projectCountResult[0]?.count ?? 0) === 0,
      hasGithubInstallation: Boolean(installation),
    };
  });

import { count, db, eq } from "@/lib/db";
import { reconcileInstallations } from "@/lib/github-installations";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { projects } from "@unkey/db/src/schema";

export const creationContext = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .query(async ({ ctx }) => {
    const [projectCountResult, installationIds] = await Promise.all([
      db
        .select({ count: count() })
        .from(projects)
        .where(eq(projects.workspaceId, ctx.workspace.id)),
      reconcileInstallations(ctx.workspace.id),
    ]);

    return {
      isFirstProject: (projectCountResult[0]?.count ?? 0) === 0,
      hasGithubInstallation: installationIds.length > 0,
    };
  });

import { db } from "@/lib/db";
import { routes } from "@/lib/navigation/routes";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { z } from "zod";

// domains.createDomain answers 409 without naming the app that holds the domain,
// which is right for a public API but leaves the dashboard unable to link there.
// Null means the domain sits in another workspace, which the caller cannot open.
export const getCustomDomainOwnerRoute = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .input(z.object({ domain: z.string().min(1) }))
  .query(async ({ input, ctx }) => {
    const row = await db.query.customDomains.findFirst({
      where: (table, { eq, and }) =>
        and(eq(table.workspaceId, ctx.workspace.id), eq(table.domain, input.domain.toLowerCase())),
      columns: { projectId: true, appId: true },
    });

    if (!row) {
      return null;
    }

    return routes.projects.apps.settings({
      workspaceSlug: ctx.workspace.slug,
      projectId: row.projectId,
      appId: row.appId,
    });
  });

import { count, db, eq, schema } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";

// The count includes all projects in the workspace. The gate in ctrl counts in
// the same way. See CountCustomDomainsByWorkspace. Thus the Limits page and the
// gate show the same number.
export const countCustomDomains = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .query(async ({ ctx }) => {
    const rows = await db
      .select({ count: count() })
      .from(schema.customDomains)
      .where(eq(schema.customDomains.workspaceId, ctx.workspace.id));

    return rows[0]?.count ?? 0;
  });

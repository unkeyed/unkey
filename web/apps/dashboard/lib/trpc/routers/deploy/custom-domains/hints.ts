import { db } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { z } from "zod";

// The API returns domainConnect only from domains.createDomain, so a reload
// would otherwise lose the one-click setup shortcut. This serves the persisted
// values so the row can offer it again.
export const listDomainConnectHints = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .input(z.object({ projectId: z.string().min(1) }))
  .query(async ({ input, ctx }) => {
    const project = await db.query.projects.findFirst({
      where: (table, { eq, and }) =>
        and(eq(table.id, input.projectId), eq(table.workspaceId, ctx.workspace.id)),
      columns: { id: true },
    });

    if (!project) {
      throw new TRPCError({ code: "NOT_FOUND", message: "Project not found" });
    }

    const rows = await db.query.customDomains.findMany({
      where: (table, { eq, and, isNotNull }) =>
        and(
          eq(table.workspaceId, ctx.workspace.id),
          eq(table.projectId, input.projectId),
          isNotNull(table.domainConnectProvider),
          isNotNull(table.domainConnectUrl),
        ),
      columns: { domain: true, domainConnectProvider: true, domainConnectUrl: true },
    });

    return rows.flatMap((row) =>
      row.domainConnectProvider && row.domainConnectUrl
        ? [{ domain: row.domain, provider: row.domainConnectProvider, url: row.domainConnectUrl }]
        : [],
    );
  });

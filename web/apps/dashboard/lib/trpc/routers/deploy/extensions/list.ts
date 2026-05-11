import { db } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { z } from "zod";
import { assertProjectInWorkspace } from "./_helpers";

export const listExtensionInstallations = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .input(z.object({ projectId: z.string().min(1) }))
  .query(async ({ input, ctx }) => {
    await assertProjectInWorkspace(input.projectId, ctx.workspace.id);

    const rows = await db.query.extensionInstallations.findMany({
      where: (table, { eq, and, isNull }) =>
        and(eq(table.projectId, input.projectId), isNull(table.deletedAt)),
      columns: {
        id: true,
        extensionSlug: true,
        instanceName: true,
        status: true,
        oauthConnected: true,
        config: true,
        lastEventAt: true,
        createdAt: true,
      },
    });

    return rows.map((row) => ({
      ...row,
      // Front-end expects ISO strings; mirrors the localStorage adapter shape.
      installedAt: new Date(row.createdAt).toISOString(),
      lastEventAt: row.lastEventAt ? new Date(row.lastEventAt).toISOString() : undefined,
    }));
  });

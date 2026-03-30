import { db } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { z } from "zod";

export const getSentinelByEnvironment = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .input(z.object({ environmentId: z.string() }))
  .query(async ({ ctx, input }) => {
    const sentinel = await db.query.sentinels.findFirst({
      where: (table, { eq, and }) =>
        and(eq(table.environmentId, input.environmentId), eq(table.workspaceId, ctx.workspace.id)),
      columns: {
        id: true,
        sentinelTierId: true,
        cpuMillicores: true,
        memoryMib: true,
      },
    });

    if (!sentinel) {
      return null;
    }

    return sentinel;
  });

import { db, eq, and, schema } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { z } from "zod";

export const listInstances = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .input(z.object({ projectId: z.string() }))
  .output(
    z.array(
      z.object({
        id: z.string(),
        status: z.string(),
        region: z.object({ name: z.string() }),
      }),
    ),
  )
  .query(async ({ ctx, input }) => {
    const instances = await db.query.instances.findMany({
      where: and(
        eq(schema.instances.projectId, input.projectId),
        eq(schema.instances.workspaceId, ctx.workspace.id),
        eq(schema.instances.status, "running"),
      ),
      columns: { id: true, status: true },
      with: {
        region: {
          columns: { name: true },
        },
      },
    });

    return instances;
  });

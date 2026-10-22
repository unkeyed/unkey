import { and, db, eq, schema } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { z } from "zod";
import { requireConnection } from "./access";
import { projectInput } from "./schemas";
import { audit } from "./write-helpers";

export const deleteAppConnection = workspaceProcedure
  .input(projectInput.extend({ id: z.string().min(1) }))
  .use(withRatelimit(ratelimit.delete))
  .mutation(async ({ ctx, input }) => {
    const connection = await requireConnection(ctx.workspace.id, input.projectId, input.id);
    await db.transaction(async (tx) => {
      await tx
        .delete(schema.appConnections)
        .where(
          and(
            eq(schema.appConnections.id, input.id),
            eq(schema.appConnections.workspaceId, ctx.workspace.id),
            eq(schema.appConnections.projectId, input.projectId),
          ),
        );
      await tx
        .delete(schema.connectionAppTargets)
        .where(eq(schema.connectionAppTargets.connectionId, connection.id));
      await audit(tx, ctx, connection.appId, `Deleted app connection ${connection.name}`);
    });
    return { id: input.id };
  });

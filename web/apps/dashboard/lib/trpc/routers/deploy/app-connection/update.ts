import { and, db, eq, schema } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { z } from "zod";
import { requireConnection } from "./access";
import { optionalName, projectInput, target, targetColumns } from "./schemas";
import { requirePrivateNetworking, requireUnusedName, validateTarget } from "./validation-helpers";
import { audit, lockPinnedTarget, mutateWithDuplicateHandling } from "./write-helpers";

export const updateAppConnection = workspaceProcedure
  .input(
    projectInput
      .extend({ id: z.string().min(1) })
      .and(optionalName)
      .and(target),
  )
  .use(withRatelimit(ratelimit.update))
  .mutation(async ({ ctx, input }) => {
    await requirePrivateNetworking();
    const connection = await requireConnection(ctx.workspace.id, input.projectId, input.id);
    await validateTarget(ctx.workspace.id, input.projectId, connection.resourceId, input);
    const name = await requireUnusedName(
      ctx.workspace.id,
      connection,
      input.name ?? connection.name,
      connection.id,
    );
    await mutateWithDuplicateHandling(async () =>
      db.transaction(async (tx) => {
        await lockPinnedTarget(tx, ctx.workspace.id, input.projectId, connection.resourceId, input);
        await tx
          .update(schema.appConnections)
          .set({ name, ...targetColumns(input), updatedAt: Date.now() })
          .where(
            and(
              eq(schema.appConnections.id, input.id),
              eq(schema.appConnections.workspaceId, ctx.workspace.id),
              eq(schema.appConnections.projectId, input.projectId),
            ),
          );
        await audit(tx, ctx, connection.appId, `Updated app connection ${name}`);
      }),
    );
    return { id: input.id, name };
  });

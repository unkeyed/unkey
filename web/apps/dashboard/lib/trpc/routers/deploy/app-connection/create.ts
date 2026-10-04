import { db, schema } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { newId } from "@unkey/id";
import { optionalName, projectInput, target, targetColumns } from "./schemas";
import { connectionEndpointsSchema } from "./validation";
import {
  pickDefaultName,
  requireUnusedName,
  validateEndpoints,
  validateTarget,
} from "./validation-helpers";
import { audit, lockPinnedTarget, mutateWithDuplicateHandling } from "./write-helpers";

export const createAppConnection = workspaceProcedure
  .input(projectInput.and(connectionEndpointsSchema).and(optionalName).and(target))
  .use(withRatelimit(ratelimit.create))
  .mutation(async ({ ctx, input }) => {
    const { targetSlug } = await validateEndpoints(ctx.workspace.id, input.projectId, input);
    await validateTarget(ctx.workspace.id, input.projectId, input.targetAppId, input);
    const name = input.name
      ? await requireUnusedName(ctx.workspace.id, input, input.name)
      : await pickDefaultName(ctx.workspace.id, input, targetSlug);
    const id = newId("connection");
    await mutateWithDuplicateHandling(async () =>
      db.transaction(async (tx) => {
        await lockPinnedTarget(tx, ctx.workspace.id, input.projectId, input.targetAppId, input);
        await tx.insert(schema.appConnections).values({
          id,
          workspaceId: ctx.workspace.id,
          projectId: input.projectId,
          appId: input.appId,
          environmentId: input.environmentId,
          resourceType: "app",
          resourceId: input.targetAppId,
          name,
          createdAt: Date.now(),
        });
        await tx.insert(schema.connectionAppTargets).values({
          connectionId: id,
          ...targetColumns(input),
        });
        await audit(tx, ctx, input.appId, `Created app connection ${name}`);
      }),
    );
    return { id, name };
  });

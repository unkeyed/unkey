import { db, schema } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { optionalName, projectInput, target, targetColumns } from "./schemas";
import { bindingEndpointsSchema } from "./validation";
import {
  pickDefaultName,
  requirePrivateNetworking,
  requireUnusedName,
  validateEndpoints,
  validateTarget,
} from "./validation-helpers";
import { audit, lockPinnedTarget, mutateWithDuplicateHandling } from "./write-helpers";

export const createAppBinding = workspaceProcedure
  .input(projectInput.and(bindingEndpointsSchema).and(optionalName).and(target))
  .use(withRatelimit(ratelimit.create))
  .mutation(async ({ ctx, input }) => {
    await requirePrivateNetworking();
    const { targetSlug } = await validateEndpoints(ctx.workspace.id, input.projectId, input);
    await validateTarget(ctx.workspace.id, input.projectId, input.targetAppId, input);
    const name = input.name
      ? await requireUnusedName(ctx.workspace.id, input, input.name)
      : await pickDefaultName(ctx.workspace.id, input, targetSlug);
    const id = `bind_${crypto.randomUUID().replaceAll("-", "").slice(0, 20)}`;
    await mutateWithDuplicateHandling(async () =>
      db.transaction(async (tx) => {
        await lockPinnedTarget(tx, ctx.workspace.id, input.projectId, input.targetAppId, input);
        await tx.insert(schema.appBindings).values({
          id,
          workspaceId: ctx.workspace.id,
          projectId: input.projectId,
          appId: input.appId,
          environmentId: input.environmentId,
          resourceType: "app",
          resourceId: input.targetAppId,
          name,
          ...targetColumns(input),
          createdAt: Date.now(),
        });
        await audit(tx, ctx, input.appId, `Created app binding ${name}`);
      }),
    );
    return { id, name };
  });

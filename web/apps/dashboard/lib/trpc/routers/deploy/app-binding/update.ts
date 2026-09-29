import { and, db, eq, schema } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { z } from "zod";
import { requireBinding } from "./access";
import { optionalName, projectInput, target, targetColumns } from "./schemas";
import { requirePrivateNetworking, requireUnusedName, validateTarget } from "./validation-helpers";
import { audit, lockPinnedTarget, mutateWithDuplicateHandling } from "./write-helpers";

export const updateAppBinding = workspaceProcedure
  .input(
    projectInput
      .extend({ id: z.string().min(1) })
      .and(optionalName)
      .and(target),
  )
  .use(withRatelimit(ratelimit.update))
  .mutation(async ({ ctx, input }) => {
    await requirePrivateNetworking();
    const binding = await requireBinding(ctx.workspace.id, input.projectId, input.id);
    await validateTarget(ctx.workspace.id, input.projectId, binding.resourceId, input);
    const name = await requireUnusedName(
      ctx.workspace.id,
      binding,
      input.name ?? binding.name,
      binding.id,
    );
    await mutateWithDuplicateHandling(async () =>
      db.transaction(async (tx) => {
        await lockPinnedTarget(tx, ctx.workspace.id, input.projectId, binding.resourceId, input);
        await tx
          .update(schema.appBindings)
          .set({ name, ...targetColumns(input), updatedAt: Date.now() })
          .where(
            and(
              eq(schema.appBindings.id, input.id),
              eq(schema.appBindings.workspaceId, ctx.workspace.id),
              eq(schema.appBindings.projectId, input.projectId),
            ),
          );
        await audit(tx, ctx, binding.appId, `Updated app binding ${name}`);
      }),
    );
    return { id: input.id, name };
  });

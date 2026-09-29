import { and, db, eq, schema } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { z } from "zod";
import { requireBinding } from "./access";
import { projectInput } from "./schemas";
import { audit } from "./write-helpers";

export const deleteAppBinding = workspaceProcedure
  .input(projectInput.extend({ id: z.string().min(1) }))
  .use(withRatelimit(ratelimit.delete))
  .mutation(async ({ ctx, input }) => {
    const binding = await requireBinding(ctx.workspace.id, input.projectId, input.id);
    await db.transaction(async (tx) => {
      await tx
        .delete(schema.appBindings)
        .where(
          and(
            eq(schema.appBindings.id, input.id),
            eq(schema.appBindings.workspaceId, ctx.workspace.id),
            eq(schema.appBindings.projectId, input.projectId),
          ),
        );
      await audit(tx, ctx, binding.appId, `Deleted app binding ${binding.name}`);
    });
    return { id: input.id };
  });

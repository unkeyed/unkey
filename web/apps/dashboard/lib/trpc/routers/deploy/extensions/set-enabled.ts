import { db, eq, schema } from "@/lib/db";
import { getProvider } from "@/lib/extensions/providers";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { z } from "zod";
import { getOwnedInstallation } from "./_helpers";

/**
 * Pause/resume an extension without uninstalling it. Provider runtimes filter
 * on `status != 'disabled'` to stop work without losing the row.
 */
export const setExtensionEnabled = workspaceProcedure
  .use(withRatelimit(ratelimit.update))
  .input(z.object({ id: z.string().min(1), enabled: z.boolean() }))
  .mutation(async ({ input, ctx }) => {
    const before = await getOwnedInstallation(input.id, ctx.workspace.id);

    const nextStatus = input.enabled ? "active" : "disabled";
    if (before.status === nextStatus) {
      return { id: input.id, status: nextStatus };
    }

    await db
      .update(schema.extensionInstallations)
      .set({ status: nextStatus })
      .where(eq(schema.extensionInstallations.id, input.id));

    await getProvider(before.extensionSlug)?.onSetEnabled?.(
      { workspaceId: ctx.workspace.id },
      before,
      input.enabled,
    );

    return { id: input.id, status: nextStatus };
  });

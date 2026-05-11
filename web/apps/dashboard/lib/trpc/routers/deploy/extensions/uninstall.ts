import { db, eq, schema } from "@/lib/db";
import { getProvider } from "@/lib/extensions/providers";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { z } from "zod";
import { getOwnedInstallation } from "./_helpers";

export const uninstallExtension = workspaceProcedure
  .use(withRatelimit(ratelimit.delete))
  .input(z.object({ id: z.string().min(1) }))
  .mutation(async ({ input, ctx }) => {
    const installation = await getOwnedInstallation(input.id, ctx.workspace.id);

    // Soft delete so audit history sticks around. Provider runtimes filter on
    // `deleted_at IS NULL` themselves — we don't push a cascade here.
    await db
      .update(schema.extensionInstallations)
      .set({ deletedAt: Date.now(), status: "disabled" })
      .where(eq(schema.extensionInstallations.id, input.id));

    await getProvider(installation.extensionSlug)?.onUninstall(
      { workspaceId: ctx.workspace.id },
      installation,
    );

    return { id: input.id };
  });

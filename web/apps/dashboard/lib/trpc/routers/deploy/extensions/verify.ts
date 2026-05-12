import { getProvider } from "@/lib/extensions/providers";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { z } from "zod";
import { extensionConfigSchema } from "./schemas";

/**
 * Wizard's "test connection" button. Dispatches to the live provider so each
 * extension owns the question of what "this is wired up correctly" actually
 * means (auth round-trip, dataset existence, webhook reachability, etc.).
 *
 * No installation row is required — the call runs against the in-flight
 * config the user is editing in the form. Preview-mode extensions return
 * `ok: true` so the wizard doesn't block their stub install.
 */
export const verifyExtension = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .input(
    z.object({
      extensionSlug: z.string().min(1).max(128),
      config: extensionConfigSchema,
    }),
  )
  .mutation(async ({ input, ctx }) => {
    const provider = getProvider(input.extensionSlug);
    if (!provider?.verify) {
      return { ok: true as const };
    }
    return provider.verify({ workspaceId: ctx.workspace.id }, input.config);
  });

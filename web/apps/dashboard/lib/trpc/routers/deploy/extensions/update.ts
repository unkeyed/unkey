import { db, eq, schema } from "@/lib/db";
import { getProvider } from "@/lib/extensions/providers";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { getOwnedInstallation } from "./_helpers";
import { extensionConfigSchema } from "./schemas";

export const updateExtensionInstallation = workspaceProcedure
  .use(withRatelimit(ratelimit.update))
  .input(
    z.object({
      id: z.string().min(1),
      instanceName: z.string().min(1).max(256).optional(),
      config: extensionConfigSchema.optional(),
    }),
  )
  .mutation(async ({ input, ctx }) => {
    const before = await getOwnedInstallation(input.id, ctx.workspace.id);

    if (input.instanceName === undefined && input.config === undefined) {
      return { id: input.id };
    }

    try {
      await db
        .update(schema.extensionInstallations)
        .set({
          ...(input.instanceName !== undefined ? { instanceName: input.instanceName } : {}),
          ...(input.config !== undefined ? { config: input.config } : {}),
        })
        .where(eq(schema.extensionInstallations.id, input.id));
    } catch (err) {
      if (err instanceof Error && err.message.includes("Duplicate entry")) {
        throw new TRPCError({
          code: "CONFLICT",
          message: `An installation named "${input.instanceName}" already exists for this extension.`,
        });
      }
      throw err;
    }

    await getProvider(before.extensionSlug)?.onUpdate?.({ workspaceId: ctx.workspace.id }, before, {
      instanceName: input.instanceName,
      config: input.config,
    });

    return { id: input.id };
  });

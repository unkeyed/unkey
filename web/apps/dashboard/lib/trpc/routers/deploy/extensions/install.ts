import { db, schema } from "@/lib/db";
import { getProvider } from "@/lib/extensions/providers";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { newId } from "@unkey/id";
import { z } from "zod";
import { assertProjectInWorkspace, getOwnedInstallation } from "./_helpers";
import { extensionConfigSchema } from "./schemas";

export const installExtension = workspaceProcedure
  .use(withRatelimit(ratelimit.create))
  .input(
    z.object({
      projectId: z.string().min(1),
      extensionSlug: z.string().min(1).max(128),
      instanceName: z.string().min(1).max(256),
      config: extensionConfigSchema,
    }),
  )
  .mutation(async ({ input, ctx }) => {
    await assertProjectInWorkspace(input.projectId, ctx.workspace.id);

    const id = newId("extensionInstallation");

    try {
      await db.insert(schema.extensionInstallations).values({
        id,
        workspaceId: ctx.workspace.id,
        projectId: input.projectId,
        extensionSlug: input.extensionSlug,
        instanceName: input.instanceName,
        config: input.config,
        status: "active",
        oauthConnected: false,
      });
    } catch (err) {
      // unique_project_extension_instance_idx collision → instance name reused.
      if (err instanceof Error && err.message.includes("Duplicate entry")) {
        throw new TRPCError({
          code: "CONFLICT",
          message: `An installation named "${input.instanceName}" already exists for this extension.`,
        });
      }
      throw err;
    }

    // Hand off to the provider (e.g. log_drains insert) for live extensions.
    const provider = getProvider(input.extensionSlug);
    if (provider) {
      const installation = await getOwnedInstallation(id, ctx.workspace.id);
      await provider.onInstall({ workspaceId: ctx.workspace.id }, installation);
    }

    return { id };
  });

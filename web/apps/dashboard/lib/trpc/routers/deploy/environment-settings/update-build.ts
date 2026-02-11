import { db } from "@/lib/db";
import { environmentBuildSettings } from "@unkey/db/src/schema";
import { z } from "zod";
import { workspaceProcedure } from "../../../trpc";

export const updateEnvironmentBuildSettings = workspaceProcedure
  .input(
    z.object({
      environmentId: z.string(),
      dockerfile: z.string().optional(),
      dockerContext: z.string().optional(),
    }),
  )
  .mutation(async ({ ctx, input }) => {
    const set: Record<string, unknown> = { updatedAt: Date.now() };
    if (input.dockerfile !== undefined) {
      set.dockerfile = input.dockerfile || "Dockerfile";
    }
    if (input.dockerContext !== undefined) {
      set.dockerContext = input.dockerContext || ".";
    }

    await db
      .insert(environmentBuildSettings)
      .values({
        workspaceId: ctx.workspace.id,
        environmentId: input.environmentId,
        dockerfile: (set.dockerfile as string) ?? "Dockerfile",
        dockerContext: (set.dockerContext as string) ?? ".",
        createdAt: Date.now(),
      })
      .onDuplicateKeyUpdate({ set });
  });

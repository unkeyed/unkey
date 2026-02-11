import { db } from "@/lib/db";
import { environmentBuildSettings } from "@unkey/db/src/schema";
import { z } from "zod";
import { workspaceProcedure } from "../../../trpc";

type BuildSettings = typeof environmentBuildSettings.$inferInsert;

export const updateEnvironmentBuildSettings = workspaceProcedure
  .input(
    z.object({
      environmentId: z.string(),
      dockerfile: z.string().optional(),
      dockerContext: z.string().optional(),
    }),
  )
  .mutation(async ({ ctx, input }) => {
    const dockerfile = input.dockerfile || "Dockerfile";
    const dockerContext = input.dockerContext || ".";

    const values: BuildSettings = {
      workspaceId: ctx.workspace.id,
      environmentId: input.environmentId,
      dockerfile,
      dockerContext,
      createdAt: Date.now(),
    };

    await db
      .insert(environmentBuildSettings)
      .values(values)
      .onDuplicateKeyUpdate({ set: values });
  });

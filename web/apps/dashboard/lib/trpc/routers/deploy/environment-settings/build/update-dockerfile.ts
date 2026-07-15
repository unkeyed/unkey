import { and, db, eq } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { appBuildSettings, environments } from "@unkey/db/src/schema";
import { z } from "zod";
import { workspaceProcedure } from "../../../../trpc";

export const updateDockerfile = workspaceProcedure
  .input(
    z.object({
      environmentId: z.string(),
      // Empty means "no Dockerfile configured": the app builds with Railpack.
      dockerfile: z.string().trim(),
    }),
  )
  .mutation(async ({ ctx, input }) => {
    const env = await db.query.environments.findFirst({
      where: and(
        eq(environments.id, input.environmentId),
        eq(environments.workspaceId, ctx.workspace.id),
      ),
      columns: { appId: true },
    });
    if (!env) {
      throw new TRPCError({ code: "NOT_FOUND", message: "Environment not found" });
    }

    // NULL is the canonical "no Dockerfile configured" representation.
    const dockerfile = input.dockerfile === "" ? null : input.dockerfile;

    await db
      .insert(appBuildSettings)
      .values({
        workspaceId: ctx.workspace.id,
        appId: env.appId,
        environmentId: input.environmentId,
        dockerfile,
        createdAt: Date.now(),
        updatedAt: Date.now(),
      })
      .onDuplicateKeyUpdate({ set: { dockerfile, updatedAt: Date.now() } });
  });

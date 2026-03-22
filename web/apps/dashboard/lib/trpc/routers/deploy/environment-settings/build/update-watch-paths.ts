import { and, db, eq } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { appBuildSettings, environments } from "@unkey/db/src/schema";
import { z } from "zod";
import { workspaceProcedure } from "../../../../trpc";

export const updateWatchPaths = workspaceProcedure
  .input(
    z.object({
      environmentId: z.string(),
      watchPaths: z.array(z.string()),
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

    await db
      .insert(appBuildSettings)
      .values({
        workspaceId: ctx.workspace.id,
        appId: env.appId,
        environmentId: input.environmentId,
        watchPaths: input.watchPaths,
        createdAt: Date.now(),
        updatedAt: Date.now(),
      })
      .onDuplicateKeyUpdate({
        set: { watchPaths: input.watchPaths, updatedAt: Date.now() },
      });
  });

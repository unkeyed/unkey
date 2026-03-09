import { and, db, eq } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { appRuntimeSettings, environments } from "@unkey/db/src/schema";
import { z } from "zod";
import { workspaceProcedure } from "../../../../trpc";

export const updateCommand = workspaceProcedure
  .input(
    z.object({
      environmentId: z.string(),
      command: z.array(z.string()),
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
      .insert(appRuntimeSettings)
      .values({
        workspaceId: ctx.workspace.id,
        appId: env.appId,
        environmentId: input.environmentId,
        command: input.command,
        sentinelConfig: "{}",
        createdAt: Date.now(),
        updatedAt: Date.now(),
      })
      .onDuplicateKeyUpdate({ set: { command: input.command, updatedAt: Date.now() } });
  });

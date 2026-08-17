import { and, db, eq } from "@/lib/db";
import { freeTierLimits } from "@/lib/limits";
import { TRPCError } from "@trpc/server";
import { appRuntimeSettings, environments, limits } from "@unkey/db/src/schema";
import { z } from "zod";
import { workspaceProcedure } from "../../../../trpc";

export const updateMemory = workspaceProcedure
  .input(
    z.object({
      environmentId: z.string(),
      memoryMib: z.number().int().min(256),
    }),
  )
  .mutation(async ({ ctx, input }) => {
    const [env, workspaceLimits] = await Promise.all([
      db.query.environments.findFirst({
        where: and(
          eq(environments.id, input.environmentId),
          eq(environments.workspaceId, ctx.workspace.id),
        ),
        columns: { appId: true },
      }),
      db.query.limits.findFirst({
        where: eq(limits.workspaceId, ctx.workspace.id),
        columns: { memoryMibMaxPerInstance: true },
      }),
    ]);
    if (!env) {
      throw new TRPCError({ code: "NOT_FOUND", message: "Environment not found" });
    }

    const maxPerInstance =
      workspaceLimits?.memoryMibMaxPerInstance ?? freeTierLimits.memoryMibMaxPerInstance;
    if (input.memoryMib > maxPerInstance) {
      throw new TRPCError({
        code: "BAD_REQUEST",
        message: `Memory per instance cannot exceed ${maxPerInstance} MiB. Contact support@unkey.com to increase it.`,
      });
    }

    await db
      .insert(appRuntimeSettings)
      .values({
        workspaceId: ctx.workspace.id,
        appId: env.appId,
        environmentId: input.environmentId,
        memoryMib: input.memoryMib,
        sentinelConfig: "{}",
        createdAt: Date.now(),
        updatedAt: Date.now(),
      })
      .onDuplicateKeyUpdate({ set: { memoryMib: input.memoryMib, updatedAt: Date.now() } });
  });

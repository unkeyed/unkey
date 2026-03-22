import { and, db, eq } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { appRuntimeSettings, environments } from "@unkey/db/src/schema";
import { z } from "zod";
import { workspaceProcedure } from "../../../../trpc";

export const updateMemory = workspaceProcedure
  .input(
    z.object({
      environmentId: z.string(),
      memoryMib: z
        .number()
        .max(
          4096,
          "Memory is limited to 4GiB during beta. Please contact support@unkey.com if you need more.",
        ),
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
        memoryMib: input.memoryMib,
        sentinelConfig: "{}",
        createdAt: Date.now(),
        updatedAt: Date.now(),
      })
      .onDuplicateKeyUpdate({ set: { memoryMib: input.memoryMib, updatedAt: Date.now() } });
  });

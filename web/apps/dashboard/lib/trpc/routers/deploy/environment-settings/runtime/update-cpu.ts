import { and, db, eq } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { appRuntimeSettings, environments, quotas } from "@unkey/db/src/schema";
import { z } from "zod";
import { workspaceProcedure } from "../../../../trpc";

export const updateCpu = workspaceProcedure
  .input(
    z.object({
      environmentId: z.string(),
      cpuMillicores: z.number().int().min(1),
    }),
  )
  .mutation(async ({ ctx, input }) => {
    const [env, quota] = await Promise.all([
      db.query.environments.findFirst({
        where: and(
          eq(environments.id, input.environmentId),
          eq(environments.workspaceId, ctx.workspace.id),
        ),
        columns: { appId: true },
      }),
      db.query.quotas.findFirst({
        where: eq(quotas.workspaceId, ctx.workspace.id),
        columns: { maxCpuMillicoresPerInstance: true },
      }),
    ]);
    if (!env) {
      throw new TRPCError({ code: "NOT_FOUND", message: "Environment not found" });
    }

    const maxPerInstance = quota?.maxCpuMillicoresPerInstance ?? 2000;
    if (input.cpuMillicores > maxPerInstance) {
      throw new TRPCError({
        code: "BAD_REQUEST",
        message: `CPU per instance cannot exceed ${maxPerInstance} millicores. Contact support@unkey.com to increase it.`,
      });
    }

    await db
      .insert(appRuntimeSettings)
      .values({
        workspaceId: ctx.workspace.id,
        appId: env.appId,
        environmentId: input.environmentId,
        cpuMillicores: input.cpuMillicores,
        sentinelConfig: "{}",
        createdAt: Date.now(),
        updatedAt: Date.now(),
      })
      .onDuplicateKeyUpdate({
        set: { cpuMillicores: input.cpuMillicores, updatedAt: Date.now() },
      });
  });

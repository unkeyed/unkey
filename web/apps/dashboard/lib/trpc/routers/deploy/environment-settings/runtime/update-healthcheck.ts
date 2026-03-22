import { and, db, eq } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { appRuntimeSettings, environments } from "@unkey/db/src/schema";
import { z } from "zod";
import { workspaceProcedure } from "../../../../trpc";

export const updateHealthcheck = workspaceProcedure
  .input(
    z.object({
      environmentId: z.string(),
      healthcheck: z
        .object({
          method: z.enum(["GET", "POST"]),
          path: z.string(),
          intervalSeconds: z.number().default(10),
          timeoutSeconds: z.number().default(5),
          failureThreshold: z.number().default(3),
          initialDelaySeconds: z.number().default(0),
        })
        .nullable(),
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
        healthcheck: input.healthcheck,
        sentinelConfig: "{}",
        createdAt: Date.now(),
        updatedAt: Date.now(),
      })
      .onDuplicateKeyUpdate({ set: { healthcheck: input.healthcheck, updatedAt: Date.now() } });
  });

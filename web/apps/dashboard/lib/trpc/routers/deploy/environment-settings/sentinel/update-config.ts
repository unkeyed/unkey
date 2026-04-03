import { and, db, eq } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { appRuntimeSettings, environments } from "@unkey/db/src/schema";
import { z } from "zod";
import { workspaceProcedure } from "../../../../trpc";

const sentinelPolicySchema = z.object({
  id: z.string(),
  name: z.string(),
  enabled: z.boolean(),
  type: z.enum(["keyauth", "ratelimit"]),
  keyauth: z.object({ keySpaceIds: z.array(z.string()) }).optional(),
  ratelimit: z.object({ limit: z.number(), windowMs: z.number() }).optional(),
});

const sentinelConfigSchema = z.object({
  policies: z.array(sentinelPolicySchema),
});

export const updateConfig = workspaceProcedure
  .input(
    z.object({
      environmentId: z.string(),
      config: sentinelConfigSchema,
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

    const serialized = JSON.stringify(input.config);

    await db
      .insert(appRuntimeSettings)
      .values({
        workspaceId: ctx.workspace.id,
        appId: env.appId,
        environmentId: input.environmentId,
        sentinelConfig: serialized,
        createdAt: Date.now(),
        updatedAt: Date.now(),
      })
      .onDuplicateKeyUpdate({ set: { sentinelConfig: serialized, updatedAt: Date.now() } });
  });

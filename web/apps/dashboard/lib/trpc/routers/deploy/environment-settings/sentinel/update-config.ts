import { and, db, eq } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { appRuntimeSettings, environments } from "@unkey/db/src/schema";
import { z } from "zod";
import { workspaceProcedure } from "../../../../trpc";

const keyLocationSchema = z.union([
  z.object({ bearer: z.object({}) }),
  z.object({ header: z.object({ name: z.string(), stripPrefix: z.string().optional() }) }),
  z.object({ queryParam: z.object({ name: z.string() }) }),
]);

const rateLimitKeySchema = z.union([
  z.object({ remoteIp: z.object({}) }),
  z.object({ header: z.object({ name: z.string() }) }),
  z.object({ authenticatedSubject: z.object({}) }),
  z.object({ path: z.object({}) }),
  z.object({ principalClaim: z.object({ claimName: z.string() }) }),
]);

const sentinelPolicySchema = z.object({
  id: z.string(),
  name: z.string(),
  enabled: z.boolean(),
  type: z.enum(["keyauth", "ratelimit", "jwt", "basicauth", "iprules", "openapi"]),
  keyauth: z
    .object({
      keySpaceIds: z.array(z.string()),
      locations: z.array(keyLocationSchema).optional(),
      permissionQuery: z.string().optional(),
    })
    .optional(),
  ratelimit: z
    .object({
      limit: z.number(),
      windowMs: z.number(),
      key: rateLimitKeySchema.optional(),
    })
    .optional(),
  jwt: z
    .object({
      jwksUri: z.string().optional(),
      issuer: z.string().optional(),
      audience: z.array(z.string()).optional(),
    })
    .optional(),
  basicauth: z
    .object({ credentials: z.array(z.object({ username: z.string(), passwordHash: z.string() })) })
    .optional(),
  iprules: z.object({ allowlist: z.array(z.string()), denylist: z.array(z.string()) }).optional(),
  openapi: z.object({ specPath: z.string() }).optional(),
  match: z.array(z.record(z.string(), z.unknown())).optional(),
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

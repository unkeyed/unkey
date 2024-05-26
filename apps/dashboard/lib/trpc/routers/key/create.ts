import { db, schema } from "@/lib/db";
import { ingestAuditLogs } from "@/lib/tinybird";
import { TRPCError } from "@trpc/server";
import { newId } from "@unkey/id";
import { newKey } from "@unkey/keys";
import { z } from "zod";
import { auth, t } from "../../trpc";

export const createKey = t.procedure
  .use(auth)
  .input(
    z.object({
      prefix: z.string().optional(),
      bytes: z.number().int().gte(1).default(16),
      keyAuthId: z.string(),
      ownerId: z.string().nullish(),
      meta: z.record(z.unknown()).optional(),
      remaining: z.number().int().positive().optional(),
      refill: z
        .object({
          interval: z.enum(["daily", "monthly"]),
          amount: z.coerce.number().int().min(1),
        })
        .optional(),
      expires: z.number().int().nullish(), // unix timestamp in milliseconds
      name: z.string().optional(),
      ratelimit: z
        .object({
          type: z.enum(["consistent", "fast"]),
          duration: z.number().int().positive(),
          limit: z.number().int().positive(),
        })
        .optional(),
      enabled: z.boolean().default(true),
      environment: z.string().optional(),
    }),
  )
  .mutation(async ({ input, ctx }) => {
    const workspace = await db.query.workspaces.findFirst({
      where: (table, { and, eq, isNull }) =>
        and(eq(table.tenantId, ctx.tenant.id), isNull(table.deletedAt)),
    });
    if (!workspace) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "workspace not found",
      });
    }

    const keyAuth = await db.query.keyAuth.findFirst({
      where: (table, { eq }) => eq(table.id, input.keyAuthId),
      with: {
        api: true,
      },
    });
    if (!keyAuth || keyAuth.workspaceId !== workspace.id) {
      throw new TRPCError({ code: "NOT_FOUND", message: "keyAuth not found" });
    }

    const keyId = newId("key");
    const { key, hash, start } = await newKey({
      prefix: input.prefix,
      byteLength: input.bytes,
    });

    await db.insert(schema.keys).values({
      id: keyId,
      keyAuthId: keyAuth.id,
      name: input.name,
      hash,
      start,
      ownerId: input.ownerId,
      meta: JSON.stringify(input.meta ?? {}),
      workspaceId: workspace.id,
      forWorkspaceId: null,
      expires: input.expires ? new Date(input.expires) : null,
      createdAt: new Date(),
      ratelimitAsync: input.ratelimit?.type === "fast",
      ratelimitLimit: input.ratelimit?.limit,
      ratelimitRefillRate: input.ratelimit?.limit,
      ratelimitDuration: input.ratelimit?.duration,
      ratelimitRefillInterval: input.ratelimit?.duration,
      ratelimitType: input.ratelimit?.type,
      remaining: input.remaining,
      refillInterval: input.refill?.interval ?? null,
      refillAmount: input.refill?.amount ?? null,
      lastRefillAt: input.refill?.interval ? new Date() : null,
      deletedAt: null,
      enabled: input.enabled,
      environment: input.environment,
    });

    await ingestAuditLogs({
      workspaceId: workspace.id,
      actor: { type: "user", id: ctx.user.id },
      event: "key.create",
      description: `Created ${keyId}`,
      resources: [
        {
          type: "key",
          id: keyId,
        },
      ],
      context: {
        location: ctx.audit.location,
        userAgent: ctx.audit.userAgent,
      },
    });

    return { keyId, key };
  });

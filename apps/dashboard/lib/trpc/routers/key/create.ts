import { db, schema } from "@/lib/db";
import { ingestAuditLogs } from "@/lib/tinybird";
import { rateLimitedProcedure, ratelimit } from "@/lib/trpc/ratelimitProcedure";
import { TRPCError } from "@trpc/server";
import { newId } from "@unkey/id";
import { newKey } from "@unkey/keys";
import { z } from "zod";

export const createKey = rateLimitedProcedure(ratelimit.create)
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
          amount: z.coerce.number().int().positive().min(1),
          dayOfMonth: z.number().int().min(1).max(31).optional(),
        })
        .optional(),
      expires: z.number().int().nullish(), // unix timestamp in milliseconds
      name: z.string().optional(),
      ratelimit: z
        .object({
          async: z.boolean(),
          duration: z.number().int().positive(),
          limit: z.number().int().positive(),
        })
        .optional(),
      enabled: z.boolean().default(true),
      environment: z.string().optional(),
    }),
  )
  .mutation(async ({ input, ctx }) => {
    const workspace = await db.query.workspaces
      .findFirst({
        where: (table, { and, eq, isNull }) =>
          and(eq(table.tenantId, ctx.tenant.id), isNull(table.deletedAt)),
      })
      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "We were unable to create a key for this API. Please contact support using support@unkey.dev.",
        });
      });
    if (!workspace) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message:
          "We are unable to find the correct workspace. Please contact support using support@unkey.dev.",
      });
    }

    const keyAuth = await db.query.keyAuth
      .findFirst({
        where: (table, { eq }) => eq(table.id, input.keyAuthId),
        with: {
          api: true,
        },
      })
      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "We were unable to create a key for this API. Please contact support using support@unkey.dev.",
        });
      });
    if (!keyAuth || keyAuth.workspaceId !== workspace.id) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message:
          "We are unable to find the correct keyAuth. Please contact support using support@unkey.dev",
      });
    }

    const keyId = newId("key");
    const { key, hash, start } = await newKey({
      prefix: input.prefix,
      byteLength: input.bytes,
    });

    await db
      .insert(schema.keys)
      .values({
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
        ratelimitAsync: input.ratelimit?.async,
        ratelimitLimit: input.ratelimit?.limit,
        ratelimitDuration: input.ratelimit?.duration,
        remaining: input.remaining,
        refillInterval: input.refill?.interval ?? null,
        refillDay: input.refill?.dayOfMonth ?? null,
        refillAmount: input.refill?.amount ?? null,
        lastRefillAt: input.refill?.interval ? new Date() : null,
        deletedAt: null,
        enabled: input.enabled,
        environment: input.environment,
      })
      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "We are unable to create the key. Please contact support using support.unkey.dev",
        });
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

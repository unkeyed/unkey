import { insertAuditLogs } from "@/lib/audit";
import { db, schema } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { newId } from "@unkey/id";
import { newKey } from "@unkey/keys";
import { z } from "zod";
import { auth, t } from "../../trpc";

export const createKey = t.procedure
  .use(auth)
  .input(
    z.object({
      prefix: z
        .string()
        .max(8, { message: "Prefixes cannot be longer than 8 characters" })
        .refine((prefix) => !prefix.includes(" "), {
          message: "Prefixes cannot contain spaces.",
        })
        .optional(),
      bytes: z
        .number()
        .int()
        .min(8, { message: "Key must be between 8 and 255 bytes long" })
        .max(255, { message: "Key must be between 8 and 255 bytes long" })
        .default(16),
      keyAuthId: z.string(),
      ownerId: z.string().nullish(),
      meta: z.record(z.unknown()).optional(),
      remaining: z.number().int().positive().optional(),
      refill: z
        .object({
          amount: z.coerce.number().int().min(1),
          refillDay: z.number().int().min(1).max(31).nullable(),
        })
        .optional(),
      expires: z.number().int().nullish(), // unix timestamp in milliseconds
      name: z.string().optional(),
      ratelimit: z
        .object({
          async: z.boolean(),
          duration: z.number().int().positive(),
          limit: z.number().nonnegative(),
        })
        .optional(),
      enabled: z.boolean().default(true),
      environment: z.string().optional(),
    }),
  )
  .mutation(async ({ input, ctx }) => {
    const keyAuth = await db.query.keyAuth
      .findFirst({
        where: (table, { and, eq }) =>
          and(eq(table.workspaceId, ctx.workspace.id), eq(table.id, input.keyAuthId)),
        with: {
          api: true,
        },
      })
      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "We were unable to create a key for this API. Please try again or contact support@unkey.dev.",
        });
      });
    if (!keyAuth) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message:
          "We are unable to find the correct keyAuth. Please try again or contact support@unkey.dev",
      });
    }

    const keyId = newId("key");
    const { key, hash, start } = await newKey({
      prefix: input.prefix,
      byteLength: input.bytes,
    });
    await db
      .transaction(async (tx) => {
        await tx.insert(schema.keys).values({
          id: keyId,
          keyAuthId: keyAuth.id,
          name: input.name,
          hash,
          start,
          ownerId: input.ownerId,
          meta: JSON.stringify(input.meta ?? {}),
          workspaceId: ctx.workspace.id,
          forWorkspaceId: null,
          expires: input.expires ? new Date(input.expires) : null,
          createdAtM: Date.now(),
          updatedAtM: null,
          ratelimitAsync: input.ratelimit?.async,
          ratelimitLimit: input.ratelimit?.limit,
          ratelimitDuration: input.ratelimit?.duration,
          remaining: input.remaining,
          refillDay: input.refill?.refillDay ?? null,
          refillAmount: input.refill?.amount ?? null,
          lastRefillAt: input.refill ? new Date() : null,
          enabled: input.enabled,
          environment: input.environment,
        });

        await insertAuditLogs(tx, ctx.workspace.auditLogBucket.id, {
          workspaceId: ctx.workspace.id,
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
      })
      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "We are unable to create the key. Please contact support using support.unkey.dev",
        });
      });

    return { keyId, key };
  });

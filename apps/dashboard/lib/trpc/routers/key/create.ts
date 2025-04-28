import { insertAuditLogs } from "@/lib/audit";
import { db, schema } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { newId } from "@unkey/id";
import { newKey } from "@unkey/keys";
import { z } from "zod";
import { requireUser, requireWorkspace, t } from "../../trpc";

export const createKey = t.procedure
  .use(requireUser)
  .use(requireWorkspace)
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
        .array(
          z.object({
            name: z.string().min(1, { message: "Name is required" }),
            refillInterval: z.coerce
              .number({
                errorMap: (issue, { defaultError }) => ({
                  message:
                    issue.code === "invalid_type"
                      ? "Duration must be greater than 0"
                      : defaultError,
                }),
              })
              .positive({ message: "Refill interval must be greater than 0" }),
            limit: z.coerce
              .number({
                errorMap: (issue, { defaultError }) => ({
                  message:
                    issue.code === "invalid_type"
                      ? "Refill limit must be greater than 0"
                      : defaultError,
                }),
              })
              .positive({ message: "Limit must be greater than 0" }),
          }),
        )
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
          remaining: input.remaining,
          refillDay: input.refill?.refillDay ?? null,
          refillAmount: input.refill?.amount ?? null,
          lastRefillAt: input.refill ? new Date() : null,
          enabled: input.enabled,
          environment: input.environment,
        });

        if (input.ratelimit?.length) {
          await tx.insert(schema.ratelimits).values(
            input.ratelimit.map((ratelimit) => ({
              id: newId("ratelimit"),
              keyId,
              duration: ratelimit.refillInterval,
              limit: ratelimit.limit,
              name: ratelimit.name,
              workspaceId: ctx.workspace.id,
              createdAt: Date.now(),
              updatedAt: null,
            })),
          );
        }

        await insertAuditLogs(tx, {
          workspaceId: ctx.workspace.id,
          actor: { type: "user", id: ctx.user.id },
          event: "key.create",
          description: `Created ${keyId}`,
          resources: [
            {
              type: "key",
              id: keyId,
              name: input.name,
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

    return { keyId, key, name: input.name };
  });

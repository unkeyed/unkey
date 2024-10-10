import { insertAuditLogs } from "@/lib/audit";
import { db, eq, schema } from "@/lib/db";
import { rateLimitedProcedure, ratelimit } from "@/lib/trpc/ratelimitProcedure";
import { TRPCError } from "@trpc/server";
import { z } from "zod";

export const updateKeyRatelimit = rateLimitedProcedure(ratelimit.update)
  .input(
    z.object({
      keyId: z.string(),
      workspaceId: z.string(),
      enabled: z.boolean(),
      ratelimitAsync: z.boolean().optional(),
      ratelimitLimit: z.number().int().positive().optional(),
      ratelimitDuration: z.number().int().positive().optional(),
    }),
  )
  .mutation(async ({ input, ctx }) => {
    const key = await db.query.keys
      .findFirst({
        where: (table, { eq, and, isNull }) =>
          and(eq(table.id, input.keyId), isNull(table.deletedAt)),
        with: {
          workspace: true,
        },
      })
      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "We were unable to update ratelimits on this key. Please try again or contact support@unkey.dev",
        });
      });
    if (!key || key.workspace.tenantId !== ctx.tenant.id) {
      throw new TRPCError({
        message:
          "We are unable to find the correct key. Please try again or contact support@unkey.dev.",
        code: "NOT_FOUND",
      });
    }

    if (input.enabled) {
      const { ratelimitAsync, ratelimitLimit, ratelimitDuration } = input;
      if (
        typeof ratelimitAsync !== "boolean" ||
        typeof ratelimitLimit !== "number" ||
        typeof ratelimitDuration !== "number"
      ) {
        throw new TRPCError({
          message: "Invalid input.",
          code: "BAD_REQUEST",
        });
      }
      await db.transaction(async (tx) => {
        await tx
          .update(schema.keys)
          .set({
            ratelimitAsync,
            ratelimitLimit,
            ratelimitDuration,
          })
          .where(eq(schema.keys.id, key.id));
        await insertAuditLogs(tx, {
          workspaceId: key.workspace.id,
          actor: {
            type: "user",
            id: ctx.user.id,
          },
          event: "key.update",
          description: `Changed ratelimit of ${key.id}`,
          resources: [
            {
              type: "key",
              id: key.id,
              meta: {
                "ratelimit.async": ratelimitAsync,
                "ratelimit.limit": ratelimitLimit,
                "ratelimit.duration": ratelimitDuration,
              },
            },
          ],
          context: {
            location: ctx.audit.location,
            userAgent: ctx.audit.userAgent,
          },
        });
      });
    } else {
      await db
        .transaction(async (tx) => {
          await tx
            .update(schema.keys)
            .set({
              ratelimitAsync: null,
              ratelimitLimit: null,
              ratelimitDuration: null,
            })
            .where(eq(schema.keys.id, key.id));

          await insertAuditLogs(tx, {
            workspaceId: key.workspace.id,
            actor: {
              type: "user",
              id: ctx.user.id,
            },
            event: "key.update",
            description: `Disabled ratelimit of ${key.id}`,
            resources: [
              {
                type: "key",
                id: key.id,
              },
            ],
            context: {
              location: ctx.audit.location,
              userAgent: ctx.audit.userAgent,
            },
          });
        })
        .catch((err) => {
          console.error(err);
          throw new TRPCError({
            code: "INTERNAL_SERVER_ERROR",
            message:
              "We were unable to update ratelimit on this key. Please try again or contact support@unkey.dev",
          });
        });
    }
  });

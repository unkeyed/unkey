import { TRPCError } from "@trpc/server";
import { z } from "zod";

import { insertAuditLogs } from "@/lib/audit";
import { db, eq, schema } from "@/lib/db";
import { ingestAuditLogsTinybird } from "@/lib/tinybird";
import { rateLimitedProcedure, ratelimit } from "@/lib/trpc/ratelimitProcedure";

export const setDefaultApiPrefix = rateLimitedProcedure(ratelimit.update)
  .input(
    z.object({
      defaultPrefix: z.string().max(8, "Prefix can be maximum of 8 charachters"),
      keyAuthId: z.string(),
      workspaceId: z.string(),
    }),
  )
  .mutation(async ({ ctx, input }) => {
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
            "We were unable to find KeyAuth. Please contact support using support@unkey.dev.",
        });
      });
    if (!keyAuth || keyAuth.workspaceId !== input.workspaceId) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message:
          "We are unable to find the correct keyAuth. Please contact support using support@unkey.dev",
      });
    }
    await db.transaction(async (tx) => {
      await tx
        .update(schema.keyAuth)
        .set({
          defaultPrefix: input.defaultPrefix,
        })
        .where(eq(schema.keyAuth.id, input.keyAuthId))
        .catch((_err) => {
          throw new TRPCError({
            code: "INTERNAL_SERVER_ERROR",
            message:
              "We were unable to update the API name. Please contact support using support@unkey.dev.",
          });
        });
      await insertAuditLogs(tx, {
        workspaceId: keyAuth.workspaceId,
        actor: {
          type: "user",
          id: ctx.user.id,
        },
        event: "api.update",
        description: `Changed ${keyAuth.workspaceId} name from ${keyAuth.defaultPrefix} to ${input.defaultPrefix}`,
        resources: [
          {
            type: "keyAuth",
            id: keyAuth.id,
          },
        ],
        context: {
          location: ctx.audit.location,
          userAgent: ctx.audit.userAgent,
        },
      });
    });
    await ingestAuditLogsTinybird({
      workspaceId: keyAuth.workspaceId,
      actor: {
        type: "user",
        id: ctx.user.id,
      },
      event: "api.update",
      description: `Changed ${keyAuth.id} name from ${keyAuth.defaultPrefix}} to ${input.defaultPrefix}`,
      resources: [
        {
          type: "keyAuth",
          id: keyAuth.id,
        },
      ],
      context: {
        location: ctx.audit.location,
        userAgent: ctx.audit.userAgent,
      },
    });
  });

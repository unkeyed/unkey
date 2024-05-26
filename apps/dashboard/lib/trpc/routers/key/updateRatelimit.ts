import { db, eq, schema } from "@/lib/db";
import { ingestAuditLogs } from "@/lib/tinybird";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { auth, t } from "../../trpc";

export const updateKeyRatelimit = t.procedure
  .use(auth)
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
    const key = await db.query.keys.findFirst({
      where: (table, { eq, and, isNull }) =>
        and(eq(table.id, input.keyId), isNull(table.deletedAt)),
      with: {
        workspace: true,
      },
    });
    if (!key || key.workspace.tenantId !== ctx.tenant.id) {
      throw new TRPCError({ message: "key not found", code: "NOT_FOUND" });
    }

    if (input.enabled) {
      const { ratelimitAsync, ratelimitLimit, ratelimitDuration } = input;
      if (
        typeof ratelimitAsync !== "boolean" ||
        typeof ratelimitLimit !== "number" ||
        typeof ratelimitDuration !== "number"
      ) {
        throw new TRPCError({ message: "invalid input", code: "BAD_REQUEST" });
      }

      await db
        .update(schema.keys)
        .set({
          ratelimitAsync,
          ratelimitType: ratelimitAsync ? "fast" : "consistent",
          ratelimitLimit,
          ratelimitRefillRate: ratelimitLimit,
          ratelimitRefillInterval: ratelimitDuration,
          ratelimitDuration,
        })
        .where(eq(schema.keys.id, key.id));

      await ingestAuditLogs({
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
    } else {
      await db
        .update(schema.keys)
        .set({
          ratelimitAsync: null,
          ratelimitType: null,
          ratelimitLimit: null,
          ratelimitRefillRate: null,
          ratelimitRefillInterval: null,
          ratelimitDuration: null,
        })
        .where(eq(schema.keys.id, key.id));

      await ingestAuditLogs({
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
    }
  });

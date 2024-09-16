import { db, eq, schema } from "@/lib/db";
import { rateLimitedProcedure, ratelimit } from "../../ratelimitProcedure";
import { ingestAuditLogs } from "@/lib/tinybird";
import { TRPCError } from "@trpc/server";
import { z } from "zod";


export const updateKeyMetadata = rateLimitedProcedure(ratelimit.update)
  .input(
    z.object({
      keyId: z.string(),
      metadata: z.string(),
    }),
  )
  .mutation(async ({ input, ctx }) => {
    let meta: unknown | null = null;

    if (input.metadata === null || input.metadata === "") {
      meta = null;
    } else {
      try {
        meta = JSON.parse(input.metadata);
      } catch (e) {
        throw new TRPCError({
          message: `The Metadata is not valid ${(e as Error).message}. Please try again.`,
          code: "BAD_REQUEST",
        });
      }
    }
    const key = await db.query.keys.findFirst({
      where: (table, { eq, isNull, and }) =>
        and(eq(table.id, input.keyId), isNull(table.deletedAt)),
      with: {
        workspace: true,
      },
    });
    if (!key || key.workspace.tenantId !== ctx.tenant.id) {
      throw new TRPCError({
        message:
          "We are unable to find the correct key. Please contact support using support@unkey.dev.",
        code: "NOT_FOUND",
      });
    }

    await db
      .update(schema.keys)
      .set({
        meta: meta ? JSON.stringify(meta) : null,
      })
      .where(eq(schema.keys.id, key.id))
      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "We are unable to update metadata on this key. Please contact support using support@unkey.dev",
        });
      });

    await ingestAuditLogs({
      workspaceId: key.workspace.id,
      actor: {
        type: "user",
        id: ctx.user.id,
      },
      event: "key.update",
      description: `Updated metadata of ${key.id}`,
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
    return true;
  });

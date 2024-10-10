import { TRPCError } from "@trpc/server";
import { z } from "zod";

import { insertAuditLogs } from "@/lib/audit";
import { db, eq, schema } from "@/lib/db";
import { rateLimitedProcedure, ratelimit } from "@/lib/trpc/ratelimitProcedure";

export const deleteApi = rateLimitedProcedure(ratelimit.delete)
  .input(
    z.object({
      apiId: z.string(),
    }),
  )
  .mutation(async ({ ctx, input }) => {
    const api = await db.query.apis
      .findFirst({
        where: (table, { eq, and, isNull }) =>
          and(eq(table.id, input.apiId), isNull(table.deletedAt)),
        with: {
          workspace: true,
        },
      })
      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "We are unable to delete this API. Please try again or contact support@unkey.dev",
        });
      });
    if (!api || api.workspace.tenantId !== ctx.tenant.id) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "The API does not exist. Please try again or contact support@unkey.dev",
      });
    }
    if (api.deleteProtection) {
      throw new TRPCError({
        code: "PRECONDITION_FAILED",
        message:
          "This API has delete protection enabled. Please disable it before deleting the API.",
      });
    }
    try {
      await db.transaction(async (tx) => {
        await tx
          .update(schema.apis)
          .set({ deletedAt: new Date() })
          .where(eq(schema.apis.id, input.apiId));
        await insertAuditLogs(tx, {
          workspaceId: api.workspaceId,
          actor: {
            type: "user",
            id: ctx.user.id,
          },
          event: "api.delete",
          description: `Deleted ${api.id}`,
          resources: [
            {
              type: "api",
              id: api.id,
            },
          ],
          context: {
            location: ctx.audit.location,
            userAgent: ctx.audit.userAgent,
          },
        });

        const keyIds = await tx.query.keys.findMany({
          where: eq(schema.keys.keyAuthId, api.keyAuthId!),
          columns: { id: true },
        });

        if (keyIds.length > 0) {
          await tx
            .update(schema.keys)
            .set({ deletedAt: new Date() })
            .where(eq(schema.keys.keyAuthId, api.keyAuthId!));
          await insertAuditLogs(
            tx,
            keyIds.map(({ id }) => ({
              workspaceId: api.workspace.id,
              actor: {
                type: "user",
                id: ctx.user.id,
              },
              event: "key.delete",
              description: `Deleted ${id} as part of the ${api.id} deletion`,
              resources: [
                {
                  type: "api",
                  id: api.id,
                },
                {
                  type: "key",
                  id: id,
                },
              ],
              context: {
                location: ctx.audit.location,
                userAgent: ctx.audit.userAgent,
              },
            })),
          );
        }
      });
    } catch (_err) {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "We are unable to delete the API. Please try again or contact support@unkey.dev",
      });
    }
  });

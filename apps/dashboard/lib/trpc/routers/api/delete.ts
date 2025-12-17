import { TRPCError } from "@trpc/server";
import { z } from "zod";

import { insertAuditLogs } from "@/lib/audit";
import { db, eq, schema } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "../../trpc";

export const deleteApi = workspaceProcedure
  .use(withRatelimit(ratelimit.delete))
  .input(
    z.object({
      apiId: z.string(),
    }),
  )
  .mutation(async ({ ctx, input }) => {
    const api = await db.query.apis
      .findFirst({
        where: (table, { eq, and, isNull }) =>
          and(
            eq(table.workspaceId, ctx.workspace.id),
            eq(table.id, input.apiId),
            isNull(table.deletedAtM),
          ),
      })
      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "We are unable to delete this API. Please try again or contact support@unkey.dev",
        });
      });
    if (!api) {
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
    const keyAuthId = api.keyAuthId;
    if (!keyAuthId) {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message:
          "This API is missing required authentication configuration and cannot be deleted. Please contact support@unkey.dev",
      });
    }
    try {
      await db.transaction(async (tx) => {
        await tx
          .update(schema.apis)
          .set({ deletedAtM: Date.now() })
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
              name: api.name,
            },
          ],
          context: {
            location: ctx.audit.location,
            userAgent: ctx.audit.userAgent,
          },
        });

        const keyIds = await tx.query.keys.findMany({
          where: eq(schema.keys.keyAuthId, keyAuthId),
          columns: { id: true },
        });

        if (keyIds.length > 0) {
          await tx
            .update(schema.keys)
            .set({ deletedAtM: Date.now() })
            .where(eq(schema.keys.keyAuthId, keyAuthId));
          await insertAuditLogs(
            tx,
            keyIds.map(({ id }) => ({
              workspaceId: ctx.workspace.id,
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
                  name: api.name,
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

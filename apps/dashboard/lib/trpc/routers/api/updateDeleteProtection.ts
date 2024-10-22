import { TRPCError } from "@trpc/server";
import { z } from "zod";

import { insertAuditLogs } from "@/lib/audit";
import { db, eq, schema } from "@/lib/db";

import { auth, t } from "../../trpc";

export const updateAPIDeleteProtection = t.procedure
  .use(auth)
  .input(
    z.object({
      apiId: z.string(),
      enabled: z.boolean(),
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
            "We were unable to update the API. Please try again or contact support@unkey.dev",
        });
      });
    if (!api || api.workspace.tenantId !== ctx.tenant.id) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message:
          "We are unable to find the correct API. Please try again or contact support@unkey.dev.",
      });
    }

    await db
      .transaction(async (tx) => {
        await tx
          .update(schema.apis)
          .set({
            deleteProtection: input.enabled,
          })
          .where(eq(schema.apis.id, input.apiId))
          .catch((_err) => {
            throw new TRPCError({
              code: "INTERNAL_SERVER_ERROR",
              message:
                "We were unable to update the API. Please try again or contact support@unkey.dev.",
            });
          });
        await insertAuditLogs(tx, {
          workspaceId: api.workspace.id,
          actor: {
            type: "user",
            id: ctx.user.id,
          },
          event: "api.update",
          description: `API ${api.name} delete protection is now ${
            input.enabled ? "enabled" : "disabled"
          }.}`,
          resources: [
            {
              type: "api",
              id: api.id,
              meta: {
                deleteProtection: input.enabled,
              },
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
            "We were unable to update the API. Please try again or contact support@unkey.dev",
        });
      });
  });

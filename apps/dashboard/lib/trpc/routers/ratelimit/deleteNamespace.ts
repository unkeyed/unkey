import { TRPCError } from "@trpc/server";
import { z } from "zod";

import { insertAuditLogs } from "@/lib/audit";
import { db, eq, schema } from "@/lib/db";
import { rateLimitedProcedure, ratelimit } from "@/lib/trpc/ratelimitProcedure";

export const deleteNamespace = rateLimitedProcedure(ratelimit.delete)
  .input(
    z.object({
      namespaceId: z.string(),
    }),
  )
  .mutation(async ({ ctx, input }) => {
    const namespace = await db.query.ratelimitNamespaces
      .findFirst({
        where: (table, { eq, and, isNull }) =>
          and(eq(table.id, input.namespaceId), isNull(table.deletedAt)),

        with: {
          workspace: {
            columns: {
              id: true,
              tenantId: true,
            },
          },
        },
      })
      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "We are unable to delete namespace. Please contact support using support@unkey.dev",
        });
      });
    if (!namespace || namespace.workspace.tenantId !== ctx.tenant.id) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message:
          "We are unable to find the correct namespace. Please contact support using support@unkey.dev.",
      });
    }

    await db.transaction(async (tx) => {
      await tx
        .update(schema.ratelimitNamespaces)
        .set({ deletedAt: new Date() })
        .where(eq(schema.ratelimitNamespaces.id, input.namespaceId));

      await insertAuditLogs(tx, {
        workspaceId: namespace.workspaceId,
        actor: {
          type: "user",
          id: ctx.user.id,
        },
        event: "ratelimitNamespace.delete",
        description: `Deleted ${namespace.id}`,
        resources: [
          {
            type: "ratelimitNamespace",
            id: namespace.id,
          },
        ],
        context: {
          location: ctx.audit.location,
          userAgent: ctx.audit.userAgent,
        },
      });

      const overrides = await tx.query.ratelimitOverrides.findMany({
        where: (table, { eq }) => eq(table.namespaceId, namespace.id),
        columns: { id: true },
      });

      if (overrides.length > 0) {
        await tx
          .update(schema.ratelimitOverrides)
          .set({ deletedAt: new Date() })
          .where(eq(schema.ratelimitOverrides.namespaceId, namespace.id))
          .catch((_err) => {
            throw new TRPCError({
              code: "INTERNAL_SERVER_ERROR",
              message:
                "We are unable to delete the namespaces. Please contact support using support@unkey.dev",
            });
          });
        await insertAuditLogs(
          tx,
          overrides.map(({ id }) => ({
            workspaceId: namespace.workspace.id,
            actor: {
              type: "user",
              id: ctx.user.id,
            },
            event: "ratelimitOverride.delete",
            description: `Deleted ${id} as part of the ${namespace.id} deletion`,
            resources: [
              {
                type: "ratelimitNamespace",
                id: namespace.id,
              },
              {
                type: "ratelimitOverride",
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
  });

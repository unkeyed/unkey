import { TRPCError } from "@trpc/server";
import { z } from "zod";

import { insertAuditLogs } from "@/lib/audit";
import { db, eq, schema } from "@/lib/db";
import { workspaceProcedure } from "../../trpc";

export const deleteNamespace = workspaceProcedure
  .input(
    z.object({
      namespaceId: z.string(),
    }),
  )
  .mutation(async ({ ctx, input }) => {
    const namespace = await db.query.ratelimitNamespaces
      .findFirst({
        where: (table, { eq, and, isNull }) =>
          and(
            eq(table.workspaceId, ctx.workspace.id),
            eq(table.id, input.namespaceId),
            isNull(table.deletedAtM),
          ),
      })
      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "We are unable to delete namespace. Please try again or contact support@unkey.com",
        });
      });
    if (!namespace) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message:
          "We are unable to find the correct namespace. Please try again or contact support@unkey.com.",
      });
    }

    await db.transaction(async (tx) => {
      await tx
        .update(schema.ratelimitNamespaces)
        .set({ deletedAtM: Date.now() })
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
            name: namespace.name,
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
          .set({ deletedAtM: Date.now() })
          .where(eq(schema.ratelimitOverrides.namespaceId, namespace.id))
          .catch((_err) => {
            throw new TRPCError({
              code: "INTERNAL_SERVER_ERROR",
              message:
                "We are unable to delete the namespaces. Please try again or contact support@unkey.com",
            });
          });
        await insertAuditLogs(
          tx,

          overrides.map(({ id }) => ({
            workspaceId: ctx.workspace.id,
            actor: {
              type: "user",
              id: ctx.user.id,
            },
            event: "ratelimit.delete_override",
            description: `Deleted ${id} as part of the ${namespace.id} deletion`,
            resources: [
              {
                type: "ratelimitNamespace",
                id: namespace.id,
                name: namespace.name,
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
        ).catch((_err) => {
          throw new TRPCError({
            code: "INTERNAL_SERVER_ERROR",
            message:
              "We are unable to delete the namespaces. Please try again or contact support@unkey.com",
          });
        });
      }
    });
  });

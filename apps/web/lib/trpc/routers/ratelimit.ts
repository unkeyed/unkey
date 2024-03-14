import { TRPCError } from "@trpc/server";
import { z } from "zod";

import { db, eq, schema } from "@/lib/db";
import { ingestAuditLogs } from "@/lib/tinybird";
import { newId } from "@unkey/id";
import { auth, t } from "../trpc";

export const ratelimitRouter = t.router({
  deleteNamespace: t.procedure
    .use(auth)
    .input(
      z.object({
        namespaceId: z.string(),
      }),
    )
    .mutation(async ({ ctx, input }) => {
      const namespace = await db.query.ratelimitNamespaces.findFirst({
        where: (table, { eq, and, isNull }) =>
          and(eq(table.id, input.namespaceId), isNull(table.deletedAt)),

        with: {
          workspace: true,
        },
      });
      if (!namespace) {
        throw new TRPCError({ code: "NOT_FOUND", message: "namespace not found" });
      }
      if (namespace.workspace.tenantId !== ctx.tenant.id) {
        throw new TRPCError({ code: "NOT_FOUND", message: "namespace not found" });
      }

      await db.transaction(async (tx) => {
        await tx
          .update(schema.ratelimitNamespaces)
          .set({ deletedAt: new Date() })
          .where(eq(schema.ratelimitNamespaces.id, input.namespaceId));

        await ingestAuditLogs({
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
        }).catch((err) => {
          tx.rollback();
          throw err;
        });

        const ratelimits = await tx.query.ratelimits.findMany({
          where: (table, { eq }) => eq(table.namespaceId, namespace.id),
          columns: { id: true },
        });

        if (ratelimits.length > 0) {
          await tx
            .update(schema.ratelimits)
            .set({ deletedAt: new Date() })
            .where(eq(schema.ratelimits.namespaceId, namespace.id));
          await ingestAuditLogs(
            ratelimits.map(({ id }) => ({
              workspaceId: namespace.workspace.id,
              actor: {
                type: "user",
                id: ctx.user.id,
              },
              event: "ratelimitIdentifier.delete",
              description: `Deleted ${id} as part of the ${namespace.id} deletion`,
              resources: [
                {
                  type: "ratelimitNamespace",
                  id: namespace.id,
                },
                {
                  type: "ratelimitIdentifier",
                  id: id,
                },
              ],
              context: {
                location: ctx.audit.location,
                userAgent: ctx.audit.userAgent,
              },
            })),
          ).catch((err) => {
            tx.rollback();
            throw err;
          });
        }
      });
    }),
  createNamespace: t.procedure
    .use(auth)
    .input(
      z.object({
        name: z.string().min(1).max(50),
      }),
    )
    .mutation(async ({ input, ctx }) => {
      const ws = await db.query.workspaces.findFirst({
        where: (table, { and, eq, isNull }) =>
          and(eq(table.tenantId, ctx.tenant.id), isNull(table.deletedAt)),
      });
      if (!ws) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "workspace not found",
        });
      }

      const namespaceId = newId("ratelimit");
      await db.insert(schema.ratelimitNamespaces).values({
        id: namespaceId,
        name: input.name,
        workspaceId: ws.id,

        createdAt: new Date(),
      });
      await ingestAuditLogs({
        workspaceId: ws.id,
        actor: {
          type: "user",
          id: ctx.user.id,
        },
        event: "ratelimitNamespace.create",
        description: `Created ${namespaceId}`,
        resources: [
          {
            type: "ratelimitNamespace",
            id: namespaceId,
          },
        ],
        context: {
          location: ctx.audit.location,
          userAgent: ctx.audit.userAgent,
        },
      });

      return {
        id: namespaceId,
      };
    }),
  updateNamespaceName: t.procedure
    .use(auth)
    .input(
      z.object({
        name: z.string().min(3, "namespace names must contain at least 3 characters"),
        namespaceId: z.string(),
        workspaceId: z.string(),
      }),
    )
    .mutation(async ({ ctx, input }) => {
      const ws = await db.query.workspaces.findFirst({
        where: (table, { and, eq, isNull }) =>
          and(eq(table.id, input.workspaceId), isNull(table.deletedAt)),
        with: {
          ratelimitNamespaces: {
            where: (table, { eq, and, isNull }) =>
              and(isNull(table.deletedAt), eq(schema.ratelimitNamespaces.id, input.namespaceId)),
          },
        },
      });

      if (!ws || ws.tenantId !== ctx.tenant.id) {
        throw new TRPCError({
          message: "workspace not found",
          code: "NOT_FOUND",
        });
      }
      const namespace = ws.ratelimitNamespaces.find((ns) => ns.id === input.namespaceId);

      if (!namespace) {
        throw new TRPCError({ message: "namespace not found", code: "NOT_FOUND" });
      }

      await db
        .update(schema.ratelimitNamespaces)
        .set({
          name: input.name,
        })
        .where(eq(schema.ratelimitNamespaces.id, input.namespaceId));
      await ingestAuditLogs({
        workspaceId: ws.id,
        actor: {
          type: "user",
          id: ctx.user.id,
        },
        event: "ratelimitNamespace.update",
        description: `Changed ${namespace.id} name from ${namespace.name} to ${input.name}`,
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
    }),
});

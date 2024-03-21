import { TRPCError } from "@trpc/server";
import { z } from "zod";

import { and, db, eq, isNull, schema, sql } from "@/lib/db";
import { ingestAuditLogs } from "@/lib/tinybird";
import { DatabaseError } from "@planetscale/database";
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

        const overrides = await tx.query.ratelimitOverrides.findMany({
          where: (table, { eq }) => eq(table.namespaceId, namespace.id),
          columns: { id: true },
        });

        if (overrides.length > 0) {
          await tx
            .update(schema.ratelimitOverrides)
            .set({ deletedAt: new Date() })
            .where(eq(schema.ratelimitOverrides.namespaceId, namespace.id));
          await ingestAuditLogs(
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

      const namespaceId = newId("ratelimitNamespace");
      try {
        await db.insert(schema.ratelimitNamespaces).values({
          id: namespaceId,
          name: input.name,
          workspaceId: ws.id,

          createdAt: new Date(),
        });
      } catch (e) {
        if (e instanceof DatabaseError && e.body.message.includes("desc = Duplicate entry")) {
          throw new TRPCError({ code: "PRECONDITION_FAILED", message: "Duplicate namespace name" });
        }
        throw e;
      }

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
  createOverride: t.procedure
    .use(auth)
    .input(
      z.object({
        namespaceId: z.string(),
        identifier: z.string(),
        limit: z.number(),
        duration: z.number(),
      }),
    )
    .mutation(async ({ input, ctx }) => {
      const namespace = await db.query.ratelimitNamespaces.findFirst({
        where: (table, { and, eq, isNull }) =>
          and(eq(table.id, input.namespaceId), isNull(table.deletedAt)),
        with: {
          workspace: {
            columns: {
              id: true,
              tenantId: true,
              features: true,
            },
          },
        },
      });
      if (!namespace || namespace.workspace.tenantId !== ctx.tenant.id) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "namespace not found",
        });
      }

      const existing = await db
        .select({ count: sql`count(*)` })
        .from(schema.ratelimitOverrides)
        .where(
          and(
            eq(schema.ratelimitOverrides.namespaceId, namespace.id),
            isNull(schema.ratelimitOverrides.deletedAt),
          ),
        )
        .then((res) => Number(res.at(0)?.count ?? 0));
      const max =
        typeof namespace.workspace.features.ratelimitOverrides === "number"
          ? namespace.workspace.features.ratelimitOverrides
          : 5;
      if (existing >= max) {
        throw new TRPCError({
          code: "FORBIDDEN",
          message: `Upgrade required, you can only override ${max} identifiers`,
        });
      }

      const id = newId("ratelimitOverride");
      await db.insert(schema.ratelimitOverrides).values({
        workspaceId: namespace.workspace.id,
        namespaceId: namespace.id,
        identifier: input.identifier,
        id,
        limit: input.limit,
        duration: input.duration,
        createdAt: new Date(),
      });
      await ingestAuditLogs({
        workspaceId: namespace.workspace.id,
        actor: {
          type: "user",
          id: ctx.user.id,
        },
        event: "ratelimitOverride.create",
        description: `Created ${input.identifier}`,
        resources: [
          {
            type: "ratelimitNamespace",
            id: input.namespaceId,
          },
          {
            type: "ratelimitOverride",
            id,
          },
        ],
        context: {
          location: ctx.audit.location,
          userAgent: ctx.audit.userAgent,
        },
      });

      return {
        id,
      };
    }),
  updateOverride: t.procedure
    .use(auth)
    .input(
      z.object({
        id: z.string(),
        limit: z.number(),
        duration: z.number(),
      }),
    )
    .mutation(async ({ ctx, input }) => {
      const override = await db.query.ratelimitOverrides.findFirst({
        where: (table, { and, eq, isNull }) => and(eq(table.id, input.id), isNull(table.deletedAt)),
        with: {
          namespace: {
            columns: {
              id: true,
            },
            with: {
              workspace: {
                columns: {
                  id: true,
                  tenantId: true,
                },
              },
            },
          },
        },
      });

      if (!override || override.namespace.workspace.tenantId !== ctx.tenant.id) {
        throw new TRPCError({
          message: "not found",
          code: "NOT_FOUND",
        });
      }

      await db
        .update(schema.ratelimitOverrides)
        .set({
          limit: input.limit,
          duration: input.duration,
          updatedAt: new Date(),
        })
        .where(eq(schema.ratelimitOverrides.id, override.id));
      await ingestAuditLogs({
        workspaceId: override.namespace.workspace.id,
        actor: {
          type: "user",
          id: ctx.user.id,
        },
        event: "ratelimitOverride.update",
        description: `Changed ${override.id} limits from ${override.limit}/${override.duration} to ${input.limit}/${input.duration}`,
        resources: [
          {
            type: "ratelimitNamespace",
            id: override.namespace.id,
          },
          {
            type: "ratelimitOverride",
            id: override.id,
          },
        ],
        context: {
          location: ctx.audit.location,
          userAgent: ctx.audit.userAgent,
        },
      });
    }),
  deleteOverride: t.procedure
    .use(auth)
    .input(
      z.object({
        id: z.string(),
      }),
    )
    .mutation(async ({ ctx, input }) => {
      const override = await db.query.ratelimitOverrides.findFirst({
        where: (table, { and, eq, isNull }) => and(eq(table.id, input.id), isNull(table.deletedAt)),
        with: {
          namespace: {
            columns: {
              id: true,
            },
            with: {
              workspace: {
                columns: {
                  id: true,
                  tenantId: true,
                },
              },
            },
          },
        },
      });

      if (!override || override.namespace.workspace.tenantId !== ctx.tenant.id) {
        throw new TRPCError({
          message: "not found",
          code: "NOT_FOUND",
        });
      }

      await db
        .delete(schema.ratelimitOverrides)
        .where(eq(schema.ratelimitOverrides.id, override.id));
      await ingestAuditLogs({
        workspaceId: override.namespace.workspace.id,
        actor: {
          type: "user",
          id: ctx.user.id,
        },
        event: "ratelimitOverride.delete",
        description: `Deleted ${override.id}`,
        resources: [
          {
            type: "ratelimitNamespace",
            id: override.namespace.id,
          },
          {
            type: "ratelimitOverride",
            id: override.id,
          },
        ],
        context: {
          location: ctx.audit.location,
          userAgent: ctx.audit.userAgent,
        },
      });
    }),
});

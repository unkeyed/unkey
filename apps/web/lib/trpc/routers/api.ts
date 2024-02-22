import { TRPCError } from "@trpc/server";
import { z } from "zod";

import { db, eq, schema } from "@/lib/db";
import { ingestAuditLogs } from "@/lib/tinybird";
import { newId } from "@unkey/id";
import { auth, t } from "../trpc";

export const apiRouter = t.router({
  delete: t.procedure
    .use(auth)
    .input(
      z.object({
        apiId: z.string(),
      }),
    )
    .mutation(async ({ ctx, input }) => {
      const api = await db.query.apis.findFirst({
        where: (table, { eq, and, isNull }) =>
          and(eq(table.id, input.apiId), isNull(table.deletedAt)),

        with: {
          workspace: true,
        },
      });
      if (!api) {
        throw new TRPCError({ code: "NOT_FOUND", message: "api not found" });
      }
      if (api.workspace.tenantId !== ctx.tenant.id) {
        throw new TRPCError({ code: "NOT_FOUND", message: "api not found" });
      }

      await db.transaction(async (tx) => {
        await tx
          .update(schema.apis)
          .set({ deletedAt: new Date() })
          .where(eq(schema.apis.id, input.apiId));

        await ingestAuditLogs({
          workspaceId: api.workspaceId,
          actor: {
            type: "user",
            id: ctx.user.id,
          },
          event: "api.delete",
          resources: [
            {
              type: "api",
              id: api.id,
            },
          ],
          context: {
            ipAddress: ctx.audit.ipAddress,
            userAgent: ctx.audit.userAgent,
          },
        }).catch((err) => {
          tx.rollback();
          throw err;
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
          await ingestAuditLogs(
            keyIds.map(({ id }) => ({
              workspaceId: api.workspace.id,
              actor: {
                type: "user",
                id: ctx.user.id,
              },
              event: "key.delete",
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
                ipAddress: ctx.audit.ipAddress,
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
  create: t.procedure
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

      const keyAuthId = newId("keyAuth");
      await db.transaction(async (tx) => {
        await tx.insert(schema.keyAuth).values({
          id: keyAuthId,
          workspaceId: ws.id,
          createdAt: new Date(),
        });
      });

      const apiId = newId("api");
      await db.transaction(async (tx) => {
        await tx.insert(schema.apis).values({
          id: apiId,
          name: input.name,
          workspaceId: ws.id,
          keyAuthId,
          authType: "key",
          ipWhitelist: null,
          createdAt: new Date(),
        });
        await ingestAuditLogs({
          workspaceId: ws.id,
          actor: {
            type: "user",
            id: ctx.user.id,
          },
          event: "api.create",
          resources: [
            {
              type: "api",
              id: apiId,
            },
          ],
          context: {
            ipAddress: ctx.audit.ipAddress,
            userAgent: ctx.audit.userAgent,
          },
        }).catch((err) => {
          tx.rollback();
          throw err;
        });
      });

      return {
        id: apiId,
      };
    }),
  updateName: t.procedure
    .use(auth)
    .input(
      z.object({
        name: z.string().min(3, "api names must contain at least 3 characters"),
        apiId: z.string(),
        workspaceId: z.string(),
      }),
    )
    .mutation(async ({ ctx, input }) => {
      const ws = await db.query.workspaces.findFirst({
        where: (table, { and, eq, isNull }) =>
          and(eq(table.id, input.workspaceId), isNull(table.deletedAt)),
        with: {
          apis: {
            where: eq(schema.apis.id, input.apiId),
          },
        },
      });

      if (!ws || ws.tenantId !== ctx.tenant.id) {
        throw new TRPCError({
          message: "workspace not found",
          code: "NOT_FOUND",
        });
      }
      const api = ws.apis.find((api) => api.id === input.apiId);

      if (!api) {
        throw new TRPCError({ message: "api not found", code: "NOT_FOUND" });
      }

      await db.transaction(async (tx) => {
        await tx
          .update(schema.apis)
          .set({
            name: input.name,
          })
          .where(eq(schema.apis.id, input.apiId));
        await ingestAuditLogs({
          workspaceId: ws.id,
          actor: {
            type: "user",
            id: ctx.user.id,
          },
          event: "api.update",
          resources: [
            {
              type: "api",
              id: api.id,
            },
          ],
          context: {
            ipAddress: ctx.audit.ipAddress,
            userAgent: ctx.audit.userAgent,
          },
        }).catch((err) => {
          tx.rollback();
          throw err;
        });
      });
    }),
  updateIpWhitelist: t.procedure
    .use(auth)
    .input(
      z.object({
        ipWhitelist: z
          .string()
          .transform((s, ctx) => {
            if (s === "") {
              return null;
            }
            const ips = s.split(/,|\n/).map((ip) => ip.trim());
            const parsedIps = z.array(z.string().ip()).safeParse(ips);
            if (!parsedIps.success) {
              ctx.addIssue(parsedIps.error.issues[0]);
              return z.NEVER;
            }
            return parsedIps.data;
          })
          .nullable(),
        apiId: z.string(),
        workspaceId: z.string(),
      }),
    )
    .mutation(async ({ input, ctx }) => {
      const ws = await db.query.workspaces.findFirst({
        where: (table, { eq, and, isNull }) =>
          and(eq(schema.workspaces.id, input.workspaceId), isNull(table.deletedAt)),
        with: {
          apis: {
            where: eq(schema.apis.id, input.apiId),
          },
        },
      });

      if (!ws || ws.tenantId !== ctx.tenant.id) {
        throw new TRPCError({
          message: "workspace not found",
          code: "NOT_FOUND",
        });
      }
      const api = ws.apis.find((api) => api.id === input.apiId);
      if (!api) {
        throw new TRPCError({ message: "api not found", code: "NOT_FOUND" });
      }

      await db.transaction(async (tx) => {
        await tx
          .update(schema.apis)
          .set({
            ipWhitelist: input.ipWhitelist === null ? null : input.ipWhitelist.join(","),
          })
          .where(eq(schema.apis.id, input.apiId));

        await ingestAuditLogs({
          workspaceId: ws.id,
          actor: {
            type: "user",
            id: ctx.user.id,
          },
          event: "api.update",
          resources: [
            {
              type: "api",
              id: api.id,
            },
          ],
          context: {
            ipAddress: ctx.audit.ipAddress,
            userAgent: ctx.audit.userAgent,
          },
        }).catch((err) => {
          tx.rollback();
          throw err;
        });
      });
    }),
});

import { TRPCError } from "@trpc/server";
import { z } from "zod";

import { db, eq, schema } from "@/lib/db";
import { env } from "@/lib/env";
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

      const { TRIGGER_API_KEY } = env();
      if (TRIGGER_API_KEY) {
        const res = await fetch("https://api.trigger.dev/api/v1/events", {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
            Authorization: `Bearer ${TRIGGER_API_KEY}`,
          },
          body: JSON.stringify({
            event: {
              name: "resources.apis.deleteApi",
              payload: {
                workspaceId: api.workspaceId,
                apiId: api.id,
                actor: {
                  type: "user",
                  id: ctx.user.id,
                },
              },
            },
          }),
        });
        if (!res.ok) {
          console.error(await res.text());
          throw new TRPCError({
            code: "INTERNAL_SERVER_ERROR",
            message: "unable to emit event to trigger.dev",
          });
        }
        await db.transaction(async (tx) => {
          await tx
            .update(schema.apis)
            .set({ state: "DELETION_IN_PROGRESS" })
            .where(eq(schema.apis.id, input.apiId));
          await tx.insert(schema.auditLogs).values({
            id: newId("auditLog"),
            time: new Date(),
            workspaceId: api.workspaceId,
            actorType: "user",
            actorId: ctx.user.id,
            event: "api.delete",
            description: `API ${api.name} marked for deletion`,
            apiId: api.id,
          });
        });
      } else {
        console.warn("TRIGGER_API_KEY not set");
        // For local development when contributors don't have access to the trigger.dev account
        await db.transaction(async (tx) => {
          await tx
            .update(schema.apis)
            .set({ deletedAt: new Date() })
            .where(eq(schema.apis.id, input.apiId));
          await tx.insert(schema.auditLogs).values({
            id: newId("auditLog"),
            time: new Date(),
            workspaceId: api.workspaceId,
            actorType: "user",
            actorId: ctx.user.id,
            event: "api.delete",
            description: `API ${api.name} deleted`,
            apiId: api.id,
          });
          await tx
            .update(schema.keys)
            .set({ deletedAt: new Date() })
            .where(eq(schema.keys.keyAuthId, api.keyAuthId!));
          // TODO
        });
      }
    }),
  create: t.procedure
    .use(auth)
    .input(
      z.object({
        name: z.string().min(1).max(50),
      }),
    )
    .mutation(async ({ input, ctx }) => {
      const workspace = await db.query.workspaces.findFirst({
        where: (table, { and, eq, isNull }) =>
          and(eq(table.tenantId, ctx.tenant.id), isNull(table.deletedAt)),
      });
      if (!workspace) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "workspace not found",
        });
      }

      const keyAuthId = newId("keyAuth");
      await db.transaction(async (tx) => {
        await tx.insert(schema.keyAuth).values({
          id: keyAuthId,
          workspaceId: workspace.id,
          createdAt: new Date(),
        });
      });

      const apiId = newId("api");
      await db.transaction(async (tx) => {
        await tx.insert(schema.apis).values({
          id: apiId,
          name: input.name,
          workspaceId: workspace.id,
          keyAuthId,
          authType: "key",
          ipWhitelist: null,
          createdAt: new Date(),
        });
        await tx.insert(schema.auditLogs).values({
          id: newId("auditLog"),
          time: new Date(),
          workspaceId: workspace.id,
          actorType: "user",
          actorId: ctx.user.id,
          event: "api.create",
          description: `API ${input.name} created`,
          apiId: apiId,
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
        await tx.insert(schema.auditLogs).values({
          id: newId("auditLog"),
          time: new Date(),
          workspaceId: ws.tenantId,
          actorType: "user",
          actorId: ctx.user.id,
          event: "api.update",
          description: `API updated from ${api.name} to ${input.name}`,
          apiId: api.id,
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
        await tx.insert(schema.auditLogs).values({
          id: newId("auditLog"),
          workspaceId: ws.id,
          apiId: api.id,
          event: "api.update",
          description: "IP whitelist updated",
          time: new Date(),
          actorType: "user",
          actorId: ctx.user.id,
        });
      });
    }),
});

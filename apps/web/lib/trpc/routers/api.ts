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
        });
      } else {
        console.warn("TRIGGER_API_KEY not set");
        // For local development when contributors don't have access to the trigger.dev account
        await db.transaction(async (tx) => {
          await tx
            .update(schema.apis)
            .set({ deletedAt: new Date() })
            .where(eq(schema.apis.id, input.apiId));
          await tx
            .update(schema.keys)
            .set({ deletedAt: new Date() })
            .where(eq(schema.keys.keyAuthId, api.keyAuthId!));
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
        throw new TRPCError({ code: "NOT_FOUND", message: "workspace not found" });
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
});

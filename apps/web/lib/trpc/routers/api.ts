import { TRPCError } from "@trpc/server";
import { z } from "zod";

import { db, eq, schema } from "@/lib/db";
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
        where: (table, { eq }) => eq(table.id, input.apiId),
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

      await db.delete(schema.apis).where(eq(schema.apis.id, input.apiId));
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
        where: (table, { eq }) => eq(table.tenantId, ctx.tenant.id),
      });
      if (!workspace) {
        throw new TRPCError({ code: "NOT_FOUND", message: "workspace not found" });
      }

      const keyAuthId = newId("keyAuth");
      await db.insert(schema.keyAuth).values({
        id: keyAuthId,
        workspaceId: workspace.id,
        createdAt: new Date(),
      });

      const apiId = newId("api");
      await db.insert(schema.apis).values({
        id: apiId,
        name: input.name,
        workspaceId: workspace.id,
        keyAuthId,
        authType: "key",
        ipWhitelist: null,
        createdAt: new Date(),
      });

      return {
        id: apiId,
      };
    }),
});

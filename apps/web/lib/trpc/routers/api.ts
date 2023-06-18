import { db, schema } from "@unkey/db";
import { TRPCError } from "@trpc/server";
import { z } from "zod";

import { t, auth } from "../trpc";
import { newId } from "@unkey/id";
import { eq } from "drizzle-orm";

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
        where: eq(schema.apis.id, input.apiId),
        with: {
          workspace: true,
        }
      });
      // Check if the API exists and if the user owns it
      if (!api || api.workspace?.tenantId !== ctx.tenant.id) {
        throw new TRPCError({ code: "NOT_FOUND", message: "api not found" });
      }

      // delete keys for the api
      await db.delete(schema.keys).where(eq(schema.keys.apiId, input.apiId));
      // delete api
      await db.delete(schema.apis).where(eq(schema.apis.id, input.apiId));
      return;
    }),
  create: t.procedure
    .use(auth)
    .input(
      z.object({
        name: z.string().min(1).max(50),
      }),
    )
    .mutation(async ({ input, ctx }) => {
      const id = newId("api");
      const workspace = await db.query.workspaces.findFirst({
        where: eq(schema.workspaces.tenantId, ctx.tenant.id),
      });
      if (!workspace) {
        throw new TRPCError({ code: "NOT_FOUND", message: "workspace not found" });
      }

      await db.insert(schema.apis).values({
        id,
        workspaceId: workspace.id,
        name: input.name,
      });

      return {
        id,
      };
    }),
});

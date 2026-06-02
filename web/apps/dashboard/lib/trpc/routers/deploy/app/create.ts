import { createAppRequestSchema } from "@/lib/collections/deploy/apps";
import { db } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { getCtrlClients } from "../../ctrl";

export const createApp = workspaceProcedure
  .input(createAppRequestSchema)
  .use(withRatelimit(ratelimit.create))
  .mutation(async ({ ctx, input }) => {
    const workspaceId = ctx.workspace.id;

    const project = await db.query.projects.findFirst({
      where: (table, { and, eq }) =>
        and(eq(table.id, input.projectId), eq(table.workspaceId, workspaceId)),
      columns: { id: true },
    });

    if (!project) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "Project not found",
      });
    }

    const existingApp = await db.query.apps.findFirst({
      where: (table, { and, eq }) =>
        and(eq(table.projectId, input.projectId), eq(table.slug, input.slug)),
      columns: { id: true },
    });

    if (existingApp) {
      throw new TRPCError({
        code: "CONFLICT",
        message: `An app with slug "${input.slug}" already exists in this project`,
      });
    }

    const ctrl = getCtrlClients();

    try {
      const response = await ctrl.app.createApp({
        workspaceId,
        projectId: input.projectId,
        name: input.name,
        slug: input.slug,
      });

      return {
        id: response.id,
      };
    } catch (err) {
      if (err instanceof TRPCError) {
        throw err;
      }
      console.error("Failed to create app:", err);
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to create app. Our team has been notified of this issue.",
      });
    }
  });

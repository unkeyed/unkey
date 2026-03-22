import { createProjectRequestSchema } from "@/lib/collections/deploy/projects";
import { db } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { getCtrlClients } from "../../ctrl";

export const createProject = workspaceProcedure
  .input(createProjectRequestSchema)
  .use(withRatelimit(ratelimit.create))
  .mutation(async ({ ctx, input }) => {
    const workspaceId = ctx.workspace.id;

    // Check if slug already exists in workspace
    const existingProject = await db.query.projects.findFirst({
      where: (table, { eq, and }) =>
        and(eq(table.workspaceId, workspaceId), eq(table.slug, input.slug)),
      columns: {
        id: true,
      },
    });

    if (existingProject) {
      throw new TRPCError({
        code: "CONFLICT",
        message: `A project with slug "${input.slug}" already exists in this workspace`,
      });
    }

    const ctrl = getCtrlClients();

    try {
      const response = await ctrl.project.createProject({
        workspaceId,
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
      console.error("Failed to create project:", err);
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to create project. Our team has been notified of this issue.",
      });
    }
  });

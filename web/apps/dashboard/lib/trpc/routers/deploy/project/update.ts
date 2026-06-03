import { createProjectRequestSchema } from "@/lib/collections/deploy/projects";
import { and, db, eq } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { projects } from "@unkey/db/src/schema";
import { z } from "zod";

const updateProjectRequestSchema = createProjectRequestSchema.extend({
  projectId: z.string().min(1, "Project ID is required"),
});

export const updateProject = workspaceProcedure
  .input(updateProjectRequestSchema)
  .use(withRatelimit(ratelimit.update))
  .mutation(async ({ ctx, input }) => {
    const workspaceId = ctx.workspace.id;

    const project = await db.query.projects.findFirst({
      where: (table, { eq, and }) =>
        and(eq(table.id, input.projectId), eq(table.workspaceId, workspaceId)),
      columns: { id: true, slug: true },
    });

    if (!project) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "Project not found",
      });
    }

    if (input.slug !== project.slug) {
      const existingProject = await db.query.projects.findFirst({
        where: (table, { eq, and }) =>
          and(eq(table.workspaceId, workspaceId), eq(table.slug, input.slug)),
        columns: { id: true },
      });

      if (existingProject) {
        throw new TRPCError({
          code: "CONFLICT",
          message: `A project with slug "${input.slug}" already exists in this workspace`,
        });
      }
    }

    try {
      await db
        .update(projects)
        .set({ name: input.name, slug: input.slug, updatedAt: Date.now() })
        .where(and(eq(projects.id, input.projectId), eq(projects.workspaceId, workspaceId)));

      return { id: project.id };
    } catch (err) {
      console.error("Failed to update project:", err);
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to update project. Our team has been notified of this issue.",
      });
    }
  });

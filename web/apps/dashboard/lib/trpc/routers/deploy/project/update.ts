import { insertAuditLogs } from "@/lib/audit";
import { createProjectRequestSchema } from "@/lib/collections/deploy/projects";
import { and, db, eq } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { projects } from "@unkey/db/src/schema";
import { z } from "zod";

// Slug is intentionally not updatable: generated deployment domains embed the
// project slug, so renaming it would orphan existing routes and free the slug
// for another project to collide with.
const updateProjectRequestSchema = createProjectRequestSchema.pick({ name: true }).extend({
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
      columns: { id: true, name: true },
    });

    if (!project) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "Project not found",
      });
    }

    try {
      await db
        .update(projects)
        .set({ name: input.name, updatedAt: Date.now() })
        .where(and(eq(projects.id, input.projectId), eq(projects.workspaceId, workspaceId)));
    } catch (err) {
      console.error("Failed to update project:", err);
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to update project. Our team has been notified of this issue.",
      });
    }

    await insertAuditLogs(db, {
      workspaceId,
      actor: { type: "user", id: ctx.user.id },
      event: "project.update",
      description: `Updated ${project.id}: name "${project.name}" -> "${input.name}"`,
      resources: [
        {
          type: "project",
          id: project.id,
          name: input.name,
        },
      ],
      context: {
        location: ctx.audit.location,
        userAgent: ctx.audit.userAgent,
      },
    });

    return { id: project.id };
  });

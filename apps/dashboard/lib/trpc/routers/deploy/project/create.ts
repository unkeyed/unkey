import { createProjectSchema } from "@/app/(app)/projects/_components/create-project/create-project.schema";
import { db, schema } from "@/lib/db";
import { ratelimit, requireUser, requireWorkspace, t, withRatelimit } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";

export const createProject = t.procedure
  .use(requireUser)
  .use(requireWorkspace)
  .input(createProjectSchema)
  .use(withRatelimit(ratelimit.create))
  .mutation(async ({ ctx, input }) => {
    try {
      // Check if slug already exists in workspace
      const existingProject = await db.query.projects.findFirst({
        where: (table, { eq, and }) =>
          and(eq(table.workspaceId, ctx.workspace.id), eq(table.slug, input.slug)),
      });

      if (existingProject) {
        throw new TRPCError({
          code: "CONFLICT",
          message: "A project with this slug already exists",
        });
      }

      const projectId = `proj_${Math.random().toString(36).substring(2)}${Date.now().toString(36)}`;
      const now = Date.now();

      await db.transaction(async (tx) => {
        // Insert new project
        await tx.insert(schema.projects).values({
          id: projectId,
          workspaceId: ctx.workspace.id,
          partitionId: "part_default", // Default partition for now
          name: input.name,
          slug: input.slug,
          gitRepositoryUrl: input.gitRepositoryUrl || null,
          deleteProtection: false,
        });

        // TODO: Add proper audit logging for projects when the audit system supports it
      });

      return {
        id: projectId,
        name: input.name,
        slug: input.slug,
        gitRepositoryUrl: input.gitRepositoryUrl,
        createdAt: now,
      };
    } catch (error) {
      if (error instanceof TRPCError) {
        throw error;
      }

      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to create project",
      });
    }
  });

import { createProjectRequestSchema } from "@/lib/collections/deploy/projects";
import { db, schema } from "@/lib/db";
import { env } from "@/lib/env";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { newId } from "@unkey/id";

export const createProject = workspaceProcedure
  .input(createProjectRequestSchema)
  .use(withRatelimit(ratelimit.create))
  .mutation(async ({ ctx, input }) => {
    const userId = ctx.user.id;
    const workspaceId = ctx.workspace.id;

    const { CTRL_URL, CTRL_API_KEY } = env();
    if (!CTRL_URL || !CTRL_API_KEY) {
      throw new TRPCError({
        code: "PRECONDITION_FAILED",
        message: "ctrl service is not configured",
      });
    }

    try {
      const workspace = await db.query.workspaces.findFirst({
        where: (table, { eq, isNull, and }) =>
          and(eq(table.id, workspaceId), isNull(table.deletedAtM)),
        columns: {
          id: true,
          orgId: true,
        },
      });

      if (!workspace) {
        console.error({
          message: "Workspace not found or deleted",
          userId,
          workspaceId,
        });
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "Workspace not found. Please verify your workspace selection and try again.",
        });
      }

      // Check if slug already exists in workspace
      const existingProject = await db.query.projects.findFirst({
        where: (table, { eq, and }) =>
          and(eq(table.workspaceId, workspaceId), eq(table.slug, input.slug)),
        columns: {
          id: true,
          slug: true,
        },
      });

      if (existingProject) {
        console.warn({
          message: "Project slug already exists in workspace",
          userId,
          workspaceId,
          projectSlug: input.slug,
          existingProjectId: existingProject.id,
        });
        throw new TRPCError({
          code: "CONFLICT",
          message: `A project with slug "${input.slug}" already exists in this workspace`,
        });
      }

      const projectId = newId("project");

      await db.transaction(async (tx) => {
        await tx.insert(schema.projects).values({
          id: projectId,
          workspaceId: ctx.workspace.id,
          name: input.name,
          slug: input.slug,
          liveDeploymentId: null,
          isRolledBack: false,
          defaultBranch: "main",
          depotProjectId: null,
          deleteProtection: false,
          createdAt: Date.now(),
          updatedAt: null,
        });

        const prodEnvId = newId("environment");
        const previewEnvId = newId("environment");

        await tx.insert(schema.environments).values([
          {
            id: prodEnvId,
            workspaceId: ctx.workspace.id,
            projectId,
            slug: "production",
            description: "Production",
            sentinelConfig: "",
            deleteProtection: false,
            createdAt: Date.now(),
            updatedAt: null,
          },
          {
            id: previewEnvId,
            workspaceId: ctx.workspace.id,
            projectId,
            slug: "preview",
            description: "Preview",
            sentinelConfig: "",
            deleteProtection: false,
            createdAt: Date.now(),
            updatedAt: null,
          },
        ]);
      });

      return {
        id: projectId,
      };
    } catch (err) {
      if (err instanceof TRPCError) {
        // Re-throw if it's already a TRPC error
        throw err;
      }

      console.error({
        message: "Unexpected error during project creation",
        userId,
        workspaceId,
        projectName: input.name,
        projectSlug: input.slug,
        error: err instanceof Error ? err.message : String(err),
        stack: err instanceof Error ? err.stack : undefined,
      });

      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to create project. Our team has been notified of this issue.",
      });
    }
  });

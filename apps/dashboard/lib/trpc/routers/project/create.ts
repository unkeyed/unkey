import { createProjectSchema } from "@/app/(app)/projects/_components/create-project/create-project.schema";
import { insertAuditLogs } from "@/lib/audit";
import { db, schema } from "@/lib/db";
import { ratelimit, requireUser, requireWorkspace, t, withRatelimit } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { newId } from "@unkey/id";

export const createProject = t.procedure
  .use(requireUser)
  .use(requireWorkspace)
  .input(createProjectSchema)
  .use(withRatelimit(ratelimit.create))
  .mutation(async ({ ctx, input }) => {
    const userId = ctx.user.id;
    const workspaceId = ctx.workspace.id;

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
      const now = Date.now();

      try {
        await db.transaction(async (tx) => {
          await tx.insert(schema.projects).values({
            id: projectId,
            workspaceId,
            name: input.name,
            slug: input.slug,
            activeDeploymentId: null,
            gitRepositoryUrl: input.gitRepositoryUrl || null,
            deleteProtection: false,
          });

          await insertAuditLogs(tx, {
            workspaceId,
            actor: {
              type: "user",
              id: userId,
            },
            event: "project.create",
            description: `Created project "${input.name}" with slug "${input.slug}"`,
            resources: [
              {
                type: "project",
                id: projectId,
                name: input.name,
              },
            ],
            context: {
              location: ctx.audit.location,
              userAgent: ctx.audit.userAgent,
            },
          });

          for (const slug of ["production", "preview"]) {
            const environmentId = newId("environment");
            await tx.insert(schema.environments).values({
              id: environmentId,
              workspaceId,
              projectId,
              slug: slug,
            });
            await insertAuditLogs(tx, {
              workspaceId,
              actor: {
                type: "user",
                id: userId,
              },
              event: "environment.create",
              description: `Created environment "${slug}" for project "${input.name}"`,
              resources: [
                {
                  type: "environment",
                  id: environmentId,
                  name: slug,
                },
              ],
              context: {
                location: ctx.audit.location,
                userAgent: ctx.audit.userAgent,
              },
            });
          }
        });
      } catch (txErr) {
        console.error({
          message: "Transaction failed during project creation",
          userId,
          workspaceId,
          projectId,
          projectSlug: input.slug,
          error: txErr instanceof Error ? txErr.message : String(txErr),
          stack: txErr instanceof Error ? txErr.stack : undefined,
        });

        throw txErr; // Re-throw to be caught by outer catch
      }

      return {
        id: projectId,
        name: input.name,
        slug: input.slug,
        gitRepositoryUrl: input.gitRepositoryUrl,
        createdAt: now,
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

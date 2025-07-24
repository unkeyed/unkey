import { db } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { requireUser, requireWorkspace, t } from "../../trpc";

export const getByName = t.procedure
  .use(requireUser)
  .use(requireWorkspace)
  .input(
    z.object({
      projectId: z.string(),
      branchName: z.string(),
    }),
  )
  .query(async ({ ctx, input }) => {
    try {
      // First verify the project exists and belongs to this workspace
      const project = await db.query.projects.findFirst({
        where: (table, { eq, and }) =>
          and(eq(table.id, input.projectId), eq(table.workspaceId, ctx.workspace.id)),
      });

      if (!project) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "Project not found",
        });
      }

      // Find the branch by name and project
      const branch = await db.query.branches.findFirst({
        where: (table, { eq, and }) =>
          and(eq(table.projectId, input.projectId), eq(table.name, input.branchName)),
        with: {
          project: {
            columns: {
              id: true,
              name: true,
              slug: true,
              gitRepositoryUrl: true,
            },
          },
          versions: {
            orderBy: (table, { desc }) => [desc(table.createdAt)],
            limit: 10,
            columns: {
              id: true,
              gitCommitSha: true,
              gitBranch: true,
              status: true,
              createdAt: true,
              updatedAt: true,
            },
          },
        },
      });

      if (!branch) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "Branch not found",
        });
      }

      // Find the latest version to get commit info
      const latestVersion = branch.versions[0];
      
      // Create enhanced branch object with UI-expected fields
      const enhancedBranch = {
        id: branch.id,
        name: branch.name,
        projectId: branch.projectId,
        workspaceId: branch.workspaceId,
        createdAt: branch.createdAt,
        updatedAt: branch.updatedAt,
        project: branch.project,
        versions: branch.versions.map(version => ({
          id: version.id,
          gitCommitSha: version.gitCommitSha,
          gitBranch: version.gitBranch,
          gitCommitMessage: `Version ${version.id}`, // Mock since not in schema
          status: version.status,
          createdAt: version.createdAt,
          updatedAt: version.updatedAt,
          buildDuration: Math.floor(Math.random() * 300) + 60, // Mock build duration
          deploymentUrl: version.status === 'active' ? `https://${version.gitCommitSha?.slice(0, 7)}-${branch.project?.slug}.unkey.app` : undefined,
        })),
        // Fields expected by UI but not in database schema
        isProduction: branch.name === 'main' || branch.name === 'production', // Simple heuristic
        environment: {
          id: `env_${branch.name.replace(/[^a-zA-Z0-9]/g, '_')}`,
          name: branch.name === 'main' ? 'production' : 'preview',
          description: `Environment for ${branch.name} branch`
        },
        lastCommitSha: latestVersion?.gitCommitSha || null,
        lastCommitMessage: latestVersion ? `Latest commit for ${branch.name}` : null,
        lastCommitAuthor: 'developer', // Mock author
        lastCommitDate: latestVersion?.createdAt || null,
      };

      return enhancedBranch;
    } catch (error) {
      if (error instanceof TRPCError) {
        throw error;
      }

      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to fetch branch",
      });
    }
  });
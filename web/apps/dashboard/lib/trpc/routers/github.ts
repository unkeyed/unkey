import { db, eq, schema } from "@/lib/db";
import { getInstallationRepositories } from "@/lib/github";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { t, workspaceProcedure } from "../trpc";

export const githubRouter = t.router({
  getInstallations: workspaceProcedure
    .input(
      z.object({
        projectId: z.string(),
      }),
    )
    .query(async ({ ctx, input }) => {
      const project = await db.query.projects.findFirst({
        where: (table, { and, eq }) =>
          and(eq(table.id, input.projectId), eq(table.workspaceId, ctx.workspace.id)),
      });

      if (!project) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "Project not found",
        });
      }

      const installations = await db.query.githubAppInstallations.findMany({
        where: eq(schema.githubAppInstallations.workspaceId, ctx.workspace.id),
      });

      const repoConnection = await db.query.githubRepoConnections.findFirst({
        where: eq(schema.githubRepoConnections.projectId, input.projectId),
      });

      return {
        installations: installations.map((i) => ({
          pk: i.pk,
          installationId: i.installationId,
        })),
        repoConnection: repoConnection
          ? {
              pk: repoConnection.pk,
              repositoryId: repoConnection.repositoryId,
              repositoryFullName: repoConnection.repositoryFullName,
            }
          : null,
      };
    }),

  listRepositories: workspaceProcedure
    .input(
      z.object({
        projectId: z.string(),
      }),
    )
    .query(async ({ ctx, input }) => {
      const project = await db.query.projects.findFirst({
        where: (table, { and, eq }) =>
          and(eq(table.id, input.projectId), eq(table.workspaceId, ctx.workspace.id)),
      });

      if (!project) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "Project not found",
        });
      }

      const installations = await db.query.githubAppInstallations.findMany({
        where: eq(schema.githubAppInstallations.workspaceId, ctx.workspace.id),
      });

      if (installations.length === 0) {
        return { repositories: [] };
      }

      const allRepos: Array<{
        id: number;
        name: string;
        fullName: string;
        private: boolean;
        htmlUrl: string;
        defaultBranch: string;
        installationId: number;
      }> = [];

      for (const installation of installations) {
        const repos = await getInstallationRepositories(installation.installationId);
        for (const repo of repos) {
          allRepos.push({
            id: repo.id,
            name: repo.name,
            fullName: repo.full_name,
            private: repo.private,
            htmlUrl: repo.html_url,
            defaultBranch: repo.default_branch,
            installationId: installation.installationId,
          });
        }
      }

      return { repositories: allRepos };
    }),

  selectRepository: workspaceProcedure
    .input(
      z.object({
        projectId: z.string(),
        repositoryId: z.number().int(),
        repositoryFullName: z.string(),
        installationId: z.number().int(),
      }),
    )
    .mutation(async ({ ctx, input }) => {
      const project = await db.query.projects.findFirst({
        where: (table, { and, eq }) =>
          and(eq(table.id, input.projectId), eq(table.workspaceId, ctx.workspace.id)),
      });

      if (!project) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "Project not found",
        });
      }

      const installation = await db.query.githubAppInstallations.findFirst({
        where: (table, { and, eq }) =>
          and(
            eq(table.installationId, input.installationId),
            eq(table.workspaceId, ctx.workspace.id),
          ),
      });

      if (!installation) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "GitHub installation not found",
        });
      }

      const existingConnection = await db.query.githubRepoConnections.findFirst({
        where: eq(schema.githubRepoConnections.projectId, input.projectId),
      });

      if (existingConnection) {
        await db
          .update(schema.githubRepoConnections)
          .set({
            installationId: input.installationId,
            repositoryId: input.repositoryId,
            repositoryFullName: input.repositoryFullName,
            updatedAt: Date.now(),
          })
          .where(eq(schema.githubRepoConnections.projectId, input.projectId));
      } else {
        await db.insert(schema.githubRepoConnections).values({
          projectId: input.projectId,
          installationId: input.installationId,
          repositoryId: input.repositoryId,
          repositoryFullName: input.repositoryFullName,
          createdAt: Date.now(),
          updatedAt: null,
        });
      }

      return { success: true };
    }),

  disconnectRepo: workspaceProcedure
    .input(
      z.object({
        projectId: z.string(),
      }),
    )
    .mutation(async ({ ctx, input }) => {
      const project = await db.query.projects.findFirst({
        where: (table, { and, eq }) =>
          and(eq(table.id, input.projectId), eq(table.workspaceId, ctx.workspace.id)),
      });

      if (!project) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "Project not found",
        });
      }

      await db
        .delete(schema.githubRepoConnections)
        .where(eq(schema.githubRepoConnections.projectId, input.projectId));

      return { success: true };
    }),

  removeInstallation: workspaceProcedure
    .input(
      z.object({
        installationId: z.number().int(),
      }),
    )
    .mutation(async ({ ctx, input }) => {
      const installation = await db.query.githubAppInstallations.findFirst({
        where: (table, { and, eq }) =>
          and(
            eq(table.installationId, input.installationId),
            eq(table.workspaceId, ctx.workspace.id),
          ),
      });

      if (!installation) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "Installation not found",
        });
      }

      await db
        .delete(schema.githubRepoConnections)
        .where(eq(schema.githubRepoConnections.installationId, input.installationId));

      await db
        .delete(schema.githubAppInstallations)
        .where(eq(schema.githubAppInstallations.installationId, input.installationId));

      return { success: true };
    }),
});

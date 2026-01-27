import { db, eq, schema } from "@/lib/db";
import { getInstallationRepositories } from "@/lib/github";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { t, workspaceProcedure } from "../trpc";

export const githubRouter = t.router({
  getInstallation: workspaceProcedure
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

      const installation = await db.query.githubAppInstallations.findFirst({
        where: eq(schema.githubAppInstallations.projectId, input.projectId),
      });

      if (!installation) {
        return { installation: null };
      }

      return {
        installation: {
          id: installation.id,
          installationId: installation.installationId,
          repositoryId: installation.repositoryId,
          repositoryFullName: installation.repositoryFullName,
        },
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

      const installation = await db.query.githubAppInstallations.findFirst({
        where: eq(schema.githubAppInstallations.projectId, input.projectId),
      });

      if (!installation) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "No GitHub installation found for this project",
        });
      }

      const repositories = await getInstallationRepositories(installation.installationId);

      return {
        repositories: repositories.map((repo) => ({
          id: repo.id,
          name: repo.name,
          fullName: repo.full_name,
          private: repo.private,
          htmlUrl: repo.html_url,
          defaultBranch: repo.default_branch,
        })),
      };
    }),

  selectRepository: workspaceProcedure
    .input(
      z.object({
        projectId: z.string(),
        repositoryId: z.string(),
        repositoryFullName: z.string(),
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
        where: eq(schema.githubAppInstallations.projectId, input.projectId),
      });

      if (!installation) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "No GitHub installation found for this project",
        });
      }

      await db
        .update(schema.githubAppInstallations)
        .set({
          repositoryId: input.repositoryId,
          repositoryFullName: input.repositoryFullName,
        })
        .where(eq(schema.githubAppInstallations.projectId, input.projectId));

      return { success: true };
    }),

  disconnect: workspaceProcedure
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
        .delete(schema.githubAppInstallations)
        .where(eq(schema.githubAppInstallations.projectId, input.projectId));

      return { success: true };
    }),
});

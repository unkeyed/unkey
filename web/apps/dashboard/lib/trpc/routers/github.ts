import { db, eq, schema } from "@/lib/db";
import { githubAppEnv } from "@/lib/env";
import { getInstallationRepositories, getRepositoryById } from "@/lib/github";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { t, workspaceProcedure } from "../trpc";

const state = z.object({
  projectId: z.string().min(1),
});

const fetchGithubContext = async (workspaceId: string, projectId: string) => {
  const project = await db.query.projects
    .findFirst({
      where: (table, { and, eq }) =>
        and(eq(table.id, projectId), eq(table.workspaceId, workspaceId)),
      columns: {
        id: true,
      },
      with: {
        githubRepoConnection: {
          columns: {
            pk: true,
            repositoryId: true,
            repositoryFullName: true,
          },
        },
        workspace: {
          columns: {
            id: true,
          },
          with: {
            githubAppInstallations: {
              columns: {
                pk: true,
                installationId: true,
              },
            },
          },
        },
      },
    })
    .catch(() => {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to load GitHub context",
      });
    });

  if (!project) {
    return null;
  }

  return {
    repoConnection: project.githubRepoConnection
      ? {
          pk: project.githubRepoConnection.pk,
          repositoryId: project.githubRepoConnection.repositoryId,
          repositoryFullName: project.githubRepoConnection.repositoryFullName,
        }
      : null,
    installations: project.workspace?.githubAppInstallations ?? [],
  };
};

const fetchProjectInstallation = async (
  workspaceId: string,
  projectId: string,
  installationId: number,
) => {
  const project = await db.query.projects
    .findFirst({
      where: (table, { and, eq }) =>
        and(eq(table.id, projectId), eq(table.workspaceId, workspaceId)),
      columns: {
        id: true,
      },
      with: {
        workspace: {
          columns: {
            id: true,
          },
          with: {
            githubAppInstallations: {
              where: (table, { eq }) => eq(table.installationId, installationId),
              columns: {
                pk: true,
              },
              limit: 1,
            },
          },
        },
      },
    })
    .catch(() => {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to load GitHub installation",
      });
    });

  if (!project) {
    return null;
  }

  return {
    installationPk: project.workspace?.githubAppInstallations?.[0]?.pk ?? null,
  };
};

export const githubRouter = t.router({
  registerInstallation: workspaceProcedure
    .input(
      z.object({
        state: z.string(),
        installationId: z.number().int(),
      }),
    )
    .mutation(async ({ ctx, input }) => {
      if (!githubAppEnv()) {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message: "GitHub App not configured",
        });
      }
      let parsedState: z.infer<typeof state> | null = null;
      try {
        const result = state.safeParse(JSON.parse(input.state));
        parsedState = result.success ? result.data : null;
      } catch {
        parsedState = null;
      }
      if (!parsedState) {
        throw new TRPCError({
          code: "BAD_REQUEST",
          message: "Invalid callback state",
        });
      }

      const projectId = parsedState.projectId;
      const projectInstallation = await fetchProjectInstallation(
        ctx.workspace.id,
        projectId,
        input.installationId,
      ).catch((err) => {
        console.error(err);
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message: "Failed to load project installation",
        });
      });

      if (!projectInstallation) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "Project not found",
        });
      }

      await db
        .insert(schema.githubAppInstallations)
        .values({
          workspaceId: ctx.workspace.id,
          installationId: input.installationId,
          createdAt: Date.now(),
          updatedAt: null,
        })
        .onDuplicateKeyUpdate({
          set: {
            updatedAt: Date.now(),
          },
        })
        .catch((err) => {
          console.error(err);
          throw new TRPCError({
            code: "INTERNAL_SERVER_ERROR",
            message: "Failed to save GitHub installation",
          });
        });

      return {
        workspaceSlug: ctx.workspace.slug,
        projectId,
      };
    }),

  getInstallations: workspaceProcedure
    .input(
      z.object({
        projectId: z.string(),
      }),
    )
    .query(async ({ ctx, input }) => {
      const githubContext = await fetchGithubContext(ctx.workspace.id, input.projectId);
      if (!githubContext) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "Project not found",
        });
      }

      return {
        installations: githubContext.installations,
        repoConnection: githubContext.repoConnection,
      };
    }),

  listRepositories: workspaceProcedure
    .input(
      z.object({
        projectId: z.string(),
      }),
    )
    .query(async ({ ctx, input }) => {
      const githubContext = await fetchGithubContext(ctx.workspace.id, input.projectId);
      if (!githubContext) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "Project not found",
        });
      }

      if (githubContext.installations.length === 0) {
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

      for (const installation of githubContext.installations) {
        const repos = await getInstallationRepositories(installation.installationId).catch(
          (err) => {
            console.error(err);
            throw new TRPCError({
              code: "INTERNAL_SERVER_ERROR",
              message: "Failed to load GitHub repositories",
            });
          },
        );
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

      allRepos.sort((a, b) => a.fullName.localeCompare(b.fullName));

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
      const projectInstallation = await fetchProjectInstallation(
        ctx.workspace.id,
        input.projectId,
        input.installationId,
      );

      if (!projectInstallation) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "Project not found",
        });
      }

      if (projectInstallation.installationPk === null) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "GitHub installation not found",
        });
      }

      const verifiedRepo = await getRepositoryById(input.installationId, input.repositoryId).catch(
        () => {
          throw new TRPCError({
            code: "INTERNAL_SERVER_ERROR",
            message: "Failed to verify GitHub repository",
          });
        },
      );

      if (!verifiedRepo) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "Repository not found in the specified GitHub installation",
        });
      }

      await db
        .insert(schema.githubRepoConnections)
        .values({
          projectId: input.projectId,
          installationId: input.installationId,
          repositoryId: verifiedRepo.id,
          repositoryFullName: verifiedRepo.full_name,
          createdAt: Date.now(),
          updatedAt: null,
        })
        .onDuplicateKeyUpdate({
          set: {
            installationId: input.installationId,
            repositoryId: verifiedRepo.id,
            repositoryFullName: verifiedRepo.full_name,
            updatedAt: Date.now(),
          },
        })
        .catch(() => {
          throw new TRPCError({
            code: "INTERNAL_SERVER_ERROR",
            message: "Failed to save GitHub repository connection",
          });
        });

      return { success: true };
    }),

  disconnectRepo: workspaceProcedure
    .input(
      z.object({
        projectId: z.string(),
      }),
    )
    .mutation(async ({ ctx, input }) => {
      const project = await db.query.projects
        .findFirst({
          where: (table, { and, eq }) =>
            and(eq(table.id, input.projectId), eq(table.workspaceId, ctx.workspace.id)),
        })
        .catch(() => {
          throw new TRPCError({
            code: "INTERNAL_SERVER_ERROR",
            message: "Failed to load project",
          });
        });

      if (!project) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "Project not found",
        });
      }

      await db
        .delete(schema.githubRepoConnections)
        .where(eq(schema.githubRepoConnections.projectId, input.projectId))
        .catch(() => {
          throw new TRPCError({
            code: "INTERNAL_SERVER_ERROR",
            message: "Failed to disconnect GitHub repository",
          });
        });

      return { success: true };
    }),

  removeInstallation: workspaceProcedure
    .input(
      z.object({
        installationId: z.number().int(),
      }),
    )
    .mutation(async ({ ctx, input }) => {
      const installation = await db.query.githubAppInstallations
        .findFirst({
          where: (table, { and, eq }) =>
            and(
              eq(table.installationId, input.installationId),
              eq(table.workspaceId, ctx.workspace.id),
            ),
        })
        .catch(() => {
          throw new TRPCError({
            code: "INTERNAL_SERVER_ERROR",
            message: "Failed to load GitHub installation",
          });
        });

      if (!installation) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "Installation not found",
        });
      }

      await db
        .delete(schema.githubRepoConnections)
        .where(eq(schema.githubRepoConnections.installationId, input.installationId))
        .catch(() => {
          throw new TRPCError({
            code: "INTERNAL_SERVER_ERROR",
            message: "Failed to remove GitHub installation",
          });
        });

      await db
        .delete(schema.githubAppInstallations)
        .where(eq(schema.githubAppInstallations.installationId, input.installationId))
        .catch(() => {
          throw new TRPCError({
            code: "INTERNAL_SERVER_ERROR",
            message: "Failed to remove GitHub installation",
          });
        });

      return { success: true };
    }),
});

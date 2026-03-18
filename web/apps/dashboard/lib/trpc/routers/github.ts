import { and, db, eq, inArray, schema } from "@/lib/db";
import { githubAppEnv } from "@/lib/env";
import {
  type BranchActivity,
  checkFileExists,
  getInstallationRepositories,
  getMostActiveBranches,
  getRepository,
  getRepositoryBranches,
  getRepositoryById,
  getRepositoryTree,
} from "@/lib/github";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { t, workspaceProcedure } from "../trpc";

const state = z.object({
  projectId: z.string().min(1),
  returnTo: z.enum(["settings"]).optional(),
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
        apps: {
          columns: { id: true, defaultBranch: true },
          with: {
            githubRepoConnection: {
              columns: {
                pk: true,
                repositoryId: true,
                repositoryFullName: true,
                installationId: true,
              },
            },
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

  // Prefer the first app that already has a github connection, otherwise pick any app
  const connectedApp = project.apps.find((a) => a.githubRepoConnection != null);
  const app = connectedApp ?? project.apps[0] ?? null;

  return {
    appId: app?.id ?? null,
    defaultBranch: app?.defaultBranch ?? "main",
    repoConnection: app?.githubRepoConnection
      ? {
          pk: app.githubRepoConnection.pk,
          repositoryId: app.githubRepoConnection.repositoryId,
          repositoryFullName: app.githubRepoConnection.repositoryFullName,
          installationId: app.githubRepoConnection.installationId,
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
  hasInstallations: workspaceProcedure.query(async ({ ctx }) => {
    const installation = await db.query.githubAppInstallations.findFirst({
      where: (table, { eq }) => eq(table.workspaceId, ctx.workspace.id),
      columns: { pk: true },
    });
    return { hasInstallation: Boolean(installation) };
  }),

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
        returnTo: parsedState.returnTo ?? null,
      };
    }),

  getRepoTree: workspaceProcedure
    .input(z.object({ projectId: z.string() }))
    .query(async ({ ctx, input }) => {
      const githubContext = await fetchGithubContext(ctx.workspace.id, input.projectId);
      if (!githubContext?.repoConnection) {
        return { tree: null };
      }

      const { repositoryFullName, installationId } = githubContext.repoConnection;
      const [owner, repo] = repositoryFullName.split("/");
      if (!owner || !repo) {
        return { tree: null };
      }

      try {
        const result = await getRepositoryTree(
          installationId,
          owner,
          repo,
          githubContext.defaultBranch,
        );

        if (result.truncated) {
          return { tree: null };
        }

        return { tree: result.tree };
      } catch {
        return { tree: null };
      }
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
        appId: githubContext.appId,
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
      if (!githubAppEnv()) {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message: "GitHub App not configured",
        });
      }

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
        pushedAt: string | null;
        language: string | null;
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
            pushedAt: repo.pushed_at,
            language: repo.language,
          });
        }
      }

      allRepos.sort((a, b) => {
        const aTime = a.pushedAt ? new Date(a.pushedAt).getTime() : 0;
        const bTime = b.pushedAt ? new Date(b.pushedAt).getTime() : 0;
        if (aTime !== bTime) {
          return bTime - aTime;
        }
        return a.fullName.localeCompare(b.fullName);
      });

      return { repositories: allRepos };
    }),

  getRepositoryDetails: workspaceProcedure
    .input(
      z.object({
        projectId: z.string(),
        installationId: z.number().int(),
        owner: z.string(),
        repo: z.string(),
        defaultBranch: z.string(),
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

      const hasInstallation = githubContext.installations.some(
        (i) => i.installationId === input.installationId,
      );
      if (!hasInstallation) {
        throw new TRPCError({
          code: "FORBIDDEN",
          message: "Installation not found for this workspace",
        });
      }

      const [treeResult, activeBranches, repoData] = await Promise.all([
        getRepositoryTree(input.installationId, input.owner, input.repo, input.defaultBranch),
        // If the events API fails, fall back to an empty list so the branch fallback logic kicks in
        getMostActiveBranches(input.installationId, input.owner, input.repo).catch(
          (): BranchActivity[] => [],
        ),
        getRepository(input.installationId, input.owner, input.repo),
      ]);

      let hasDockerfile: boolean;
      if (treeResult.truncated) {
        // Tree was truncated by GitHub — scanning the partial tree may miss files.
        // Fall back to a targeted existence check for the root Dockerfile.
        hasDockerfile = await checkFileExists(
          input.installationId,
          input.owner,
          input.repo,
          input.defaultBranch,
          "Dockerfile",
        );
      } else {
        hasDockerfile = treeResult.tree.some(
          (entry) => entry.type === "blob" && entry.path.split("/").pop() === "Dockerfile",
        );
      }

      let branches: Array<{ name: string; lastPushDate: string | null }>;

      if (activeBranches.length > 0) {
        branches = activeBranches.map((b) => ({ name: b.name, lastPushDate: b.lastPushDate }));
        // Ensure the default branch is always included
        if (!branches.some((b) => b.name === input.defaultBranch)) {
          const defaultEntry = activeBranches.find((b) => b.name === input.defaultBranch);
          branches.unshift({
            name: input.defaultBranch,
            lastPushDate: defaultEntry?.lastPushDate ?? null,
          });
        }
      } else {
        // Fallback: no recent events, use alphabetical branches (capped at 10)
        const fallbackBranches = await getRepositoryBranches(
          input.installationId,
          input.owner,
          input.repo,
          10,
        );
        branches = fallbackBranches.map((b) => ({ name: b.name, lastPushDate: null }));
      }

      return {
        hasDockerfile,
        branches,
        pushedAt: repoData.pushed_at,
      };
    }),

  selectRepository: workspaceProcedure
    .input(
      z.object({
        projectId: z.string(),
        appId: z.string().optional(),
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

      // Resolve appId: use provided value or find the default app for the project
      let appId = input.appId;
      if (!appId) {
        const app = await db.query.apps.findFirst({
          where: (table, { eq, and }) =>
            and(eq(table.projectId, input.projectId), eq(table.slug, "default")),
          columns: { id: true },
        });
        if (!app) {
          throw new TRPCError({
            code: "NOT_FOUND",
            message: "No default app found for this project",
          });
        }
        appId = app.id;
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
          appId,
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

      // Persist the repo's default branch to the app so branch→environment
      // resolution uses the actual GitHub default instead of hardcoded "main".
      if (verifiedRepo.default_branch) {
        await db
          .update(schema.apps)
          .set({ defaultBranch: verifiedRepo.default_branch, updatedAt: Date.now() })
          .where(eq(schema.apps.id, appId));
      }

      return { success: true };
    }),

  disconnectRepo: workspaceProcedure
    .input(
      z.object({
        appId: z.string(),
      }),
    )
    .mutation(async ({ ctx, input }) => {
      const app = await db.query.apps
        .findFirst({
          where: (table, { and, eq }) =>
            and(eq(table.id, input.appId), eq(table.workspaceId, ctx.workspace.id)),
        })
        .catch(() => {
          throw new TRPCError({
            code: "INTERNAL_SERVER_ERROR",
            message: "Failed to load app",
          });
        });

      if (!app) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "App not found",
        });
      }

      await db
        .delete(schema.githubRepoConnections)
        .where(eq(schema.githubRepoConnections.appId, input.appId))
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
        .where(
          and(
            eq(schema.githubRepoConnections.installationId, input.installationId),
            inArray(
              schema.githubRepoConnections.projectId,
              db
                .select({ id: schema.projects.id })
                .from(schema.projects)
                .where(eq(schema.projects.workspaceId, ctx.workspace.id)),
            ),
          ),
        )
        .catch(() => {
          throw new TRPCError({
            code: "INTERNAL_SERVER_ERROR",
            message: "Failed to remove GitHub installation",
          });
        });

      await db
        .delete(schema.githubAppInstallations)
        .where(
          and(
            eq(schema.githubAppInstallations.installationId, input.installationId),
            eq(schema.githubAppInstallations.workspaceId, ctx.workspace.id),
          ),
        )
        .catch(() => {
          throw new TRPCError({
            code: "INTERNAL_SERVER_ERROR",
            message: "Failed to remove GitHub installation",
          });
        });

      return { success: true };
    }),
});

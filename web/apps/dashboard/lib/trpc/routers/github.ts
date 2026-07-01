import crypto from "node:crypto";
import { and, db, eq, schema } from "@/lib/db";
import { githubAppEnv, githubOAuthEnv } from "@/lib/env";
import {
  type BranchActivity,
  MAX_BRANCHES,
  exchangeInstallationOAuthCode,
  getInstallationRepositories,
  getMostActiveBranches,
  getRepository,
  getRepositoryBranches,
  getRepositoryById,
  getRepositoryTree,
  searchBranchesByPrefix,
  userCanAccessInstallation,
} from "@/lib/github";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { t, workspaceProcedure } from "../trpc";

const STATE_TTL_MS = 15 * 60 * 1000;

// State payload signed and handed to GitHub's install URL. The signature
// binds the state to a specific user + workspace + project so the callback
// cannot be replayed across sessions or used by an attacker who phishes
// a logged-in victim into hitting /integrations/github/callback?state=...
const signedStatePayload = z.object({
  projectId: z.string().min(1),
  appId: z.string().min(1),
  returnTo: z.enum(["settings"]).optional(),
  workspaceId: z.string().min(1),
  userId: z.string().min(1),
  nonce: z.string().min(1),
  exp: z.number().int().positive(),
});

const signedState = signedStatePayload.extend({ sig: z.string().min(1) });

type SignedStatePayload = z.infer<typeof signedStatePayload>;

const stateSigningKey = (): Buffer | null => {
  const env = githubAppEnv();
  if (!env) {
    return null;
  }
  // The GitHub App's RSA private key already exists as a long-lived server
  // secret; derive a separate HMAC key from it so the signing key never
  // leaves the server and rotates with the GitHub App credentials.
  return crypto
    .createHash("sha256")
    .update(`unkey-github-install-state:${env.UNKEY_GITHUB_PRIVATE_KEY_PEM}`)
    .digest();
};

const stableStringify = (payload: SignedStatePayload): string =>
  JSON.stringify(payload, Object.keys(payload).sort());

const signState = (payload: SignedStatePayload): string => {
  const key = stateSigningKey();
  if (!key) {
    throw new TRPCError({
      code: "INTERNAL_SERVER_ERROR",
      message: "GitHub App not configured",
    });
  }
  const sig = crypto.createHmac("sha256", key).update(stableStringify(payload)).digest("base64url");
  return JSON.stringify({ ...payload, sig });
};

const verifyState = (raw: string): SignedStatePayload | null => {
  let parsed: unknown;
  try {
    parsed = JSON.parse(raw);
  } catch {
    return null;
  }
  const result = signedState.safeParse(parsed);
  if (!result.success) {
    return null;
  }
  const { sig, ...payload } = result.data;
  const key = stateSigningKey();
  if (!key) {
    return null;
  }
  const expected = crypto
    .createHmac("sha256", key)
    .update(stableStringify(payload))
    .digest("base64url");
  const a = Buffer.from(sig);
  const b = Buffer.from(expected);
  if (a.length !== b.length || !crypto.timingSafeEqual(a, b)) {
    return null;
  }
  if (payload.exp < Date.now()) {
    return null;
  }
  return payload;
};

const fetchGithubContext = async (workspaceId: string, projectId: string, appId?: string) => {
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

  // Scope to the requested app when given. Otherwise prefer the first app that
  // already has a github connection, falling back to any app.
  const app = appId
    ? (project.apps.find((a) => a.id === appId) ?? null)
    : (project.apps.find((a) => a.githubRepoConnection != null) ?? project.apps[0] ?? null);

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

  prepareInstallation: workspaceProcedure
    .input(
      z.object({
        projectId: z.string().min(1),
        appId: z.string().min(1),
        returnTo: z.enum(["settings"]).optional(),
      }),
    )
    .mutation(async ({ ctx, input }) => {
      if (!githubAppEnv()) {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message: "GitHub App not configured",
        });
      }

      // Verify the project belongs to the calling workspace before issuing a
      // signed state. Without this, an attacker could mint a state for an
      // arbitrary project id.
      const project = await db.query.projects.findFirst({
        where: (table, { and, eq }) =>
          and(eq(table.id, input.projectId), eq(table.workspaceId, ctx.workspace.id)),
        columns: { id: true },
      });
      if (!project) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "Project not found",
        });
      }

      return {
        state: signState({
          projectId: input.projectId,
          appId: input.appId,
          returnTo: input.returnTo,
          workspaceId: ctx.workspace.id,
          userId: ctx.user.id,
          nonce: crypto.randomBytes(16).toString("base64url"),
          exp: Date.now() + STATE_TTL_MS,
        }),
      };
    }),

  registerInstallation: workspaceProcedure
    .input(
      z.object({
        state: z.string(),
        installationId: z.number().int(),
        // OAuth `code` returned alongside installation_id when the GitHub App
        // requests user authorization during installation. Used to prove the
        // caller can actually access the supplied installation on GitHub.
        code: z.string().min(1),
      }),
    )
    .mutation(async ({ ctx, input }) => {
      if (!githubAppEnv() || !githubOAuthEnv()) {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message: "GitHub App not configured",
        });
      }

      // The state must be a server-signed token bound to the calling user
      // and workspace. This prevents two attacks:
      //  1) An attacker claiming a victim's installation id (sequential
      //     integers visible in webhooks/URLs) by forging a JSON state.
      //  2) Phishing a logged-in victim into POSTing this mutation under
      //     the attacker's chosen state — the userId/workspaceId binding
      //     in the signature would not match the victim's session.
      const parsedState = verifyState(input.state);
      if (
        !parsedState ||
        parsedState.workspaceId !== ctx.workspace.id ||
        parsedState.userId !== ctx.user.id
      ) {
        throw new TRPCError({
          code: "BAD_REQUEST",
          message: "Invalid callback state",
        });
      }

      const projectId = parsedState.projectId;

      // Verify the caller actually owns/can access this installation on
      // GitHub before binding it. installationId comes straight from the
      // OAuth callback query string and is a small, enumerable, sequential
      // integer exposed in webhooks/URLs. The signed state only proves who the
      // caller is, not that they performed this installation, and the
      // existing-binding check below only blocks re-claiming an installation
      // already registered to another workspace. Without this step a caller
      // could bind a victim org's unregistered installation to their own
      // workspace and read its private repos via the app-minted access token.
      // We exchange the OAuth code for a user-to-server token and confirm the
      // authenticated GitHub identity can see this installation.
      let userToken: string;
      try {
        userToken = await exchangeInstallationOAuthCode(input.code);
      } catch (err) {
        console.error(err);
        throw new TRPCError({
          code: "BAD_REQUEST",
          message: "Invalid GitHub authorization",
        });
      }

      let canAccessInstallation: boolean;
      try {
        canAccessInstallation = await userCanAccessInstallation(userToken, input.installationId);
      } catch (err) {
        console.error(err);
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message: "Failed to verify GitHub installation ownership",
        });
      }

      if (!canAccessInstallation) {
        throw new TRPCError({
          code: "FORBIDDEN",
          message: "You do not have access to this GitHub installation",
        });
      }

      // Refuse to bind the same installation id to multiple workspaces. An
      // attacker who already owns a workspace could otherwise re-register a
      // victim org's installation id under their workspace, and use the
      // resulting Unkey-minted access token to read the victim's repos.
      const existing = await db.query.githubAppInstallations.findFirst({
        where: (table, { eq }) => eq(table.installationId, input.installationId),
        columns: { workspaceId: true },
      });
      if (existing && existing.workspaceId !== ctx.workspace.id) {
        throw new TRPCError({
          code: "CONFLICT",
          message: "GitHub installation is already bound to another workspace",
        });
      }

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
        appId: parsedState.appId,
        returnTo: parsedState.returnTo ?? null,
      };
    }),

  getRepoTree: workspaceProcedure
    .input(z.object({ projectId: z.string(), appId: z.string().min(1) }))
    .query(async ({ ctx, input }) => {
      const githubContext = await fetchGithubContext(
        ctx.workspace.id,
        input.projectId,
        input.appId,
      );
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

        return { tree: result.tree, branch: githubContext.defaultBranch };
      } catch {
        return { tree: null, branch: githubContext.defaultBranch };
      }
    }),

  getInstallations: workspaceProcedure
    .input(
      z.object({
        projectId: z.string(),
        appId: z.string().min(1),
      }),
    )
    .query(async ({ ctx, input }) => {
      const githubContext = await fetchGithubContext(
        ctx.workspace.id,
        input.projectId,
        input.appId,
      );
      if (!githubContext) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "Project not found",
        });
      }

      return {
        appId: githubContext.appId,
        defaultBranch: githubContext.defaultBranch,
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

      const [repoData, activeBranches, alphabeticalBranches] = await Promise.all([
        getRepository(input.installationId, input.owner, input.repo),
        getMostActiveBranches(input.installationId, input.owner, input.repo).catch(
          (): BranchActivity[] => [],
        ),
        getRepositoryBranches(input.installationId, input.owner, input.repo, MAX_BRANCHES),
      ]);

      const activityMap = new Map(activeBranches.map((b) => [b.name, b.lastPushDate]));
      const seen = new Set<string>();
      const branches: Array<{ name: string; lastPushDate: string | null }> = [];

      for (const b of activeBranches) {
        if (!seen.has(b.name)) {
          seen.add(b.name);
          branches.push({ name: b.name, lastPushDate: b.lastPushDate });
        }
      }

      for (const b of alphabeticalBranches) {
        if (!seen.has(b.name) && branches.length < MAX_BRANCHES) {
          seen.add(b.name);
          branches.push({ name: b.name, lastPushDate: activityMap.get(b.name) ?? null });
        }
      }

      if (!seen.has(input.defaultBranch)) {
        branches.unshift({ name: input.defaultBranch, lastPushDate: null });
      }

      return {
        branches,
        pushedAt: repoData.pushed_at,
      };
    }),

  searchBranches: workspaceProcedure
    .input(
      z.object({
        projectId: z.string(),
        installationId: z.number().int(),
        owner: z.string(),
        repo: z.string(),
        query: z.string().min(1).max(200),
      }),
    )
    .query(async ({ ctx, input }) => {
      const githubContext = await fetchGithubContext(ctx.workspace.id, input.projectId);
      if (!githubContext) {
        throw new TRPCError({ code: "NOT_FOUND", message: "Project not found" });
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
      const branches = await searchBranchesByPrefix(
        input.installationId,
        input.owner,
        input.repo,
        input.query,
      );
      return { branches };
    }),

  selectRepository: workspaceProcedure
    .input(
      z.object({
        projectId: z.string(),
        appId: z.string().optional(),
        repositoryId: z.number().int(),
        repositoryFullName: z.string(),
        installationId: z.number().int(),
        selectedBranch: z.string().optional(),
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
      let appId: string;
      if (input.appId) {
        const requestedAppId = input.appId;
        const app = await db.query.apps.findFirst({
          where: (table, { eq, and }) =>
            and(
              eq(table.id, requestedAppId),
              eq(table.workspaceId, ctx.workspace.id),
              eq(table.projectId, input.projectId),
            ),
          columns: { id: true },
        });
        if (!app) {
          throw new TRPCError({
            code: "NOT_FOUND",
            message: "App not found for this project",
          });
        }
        appId = app.id;
      } else {
        const app = await db.query.apps.findFirst({
          where: (table, { eq, and }) =>
            and(
              eq(table.projectId, input.projectId),
              eq(table.workspaceId, ctx.workspace.id),
              eq(table.slug, "default"),
            ),
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
          workspaceId: ctx.workspace.id,
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

      const branchToStore = input.selectedBranch ?? verifiedRepo.default_branch;
      if (branchToStore) {
        await db
          .update(schema.apps)
          .set({ defaultBranch: branchToStore, updatedAt: Date.now() })
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

  updateDefaultBranch: workspaceProcedure
    .input(
      z.object({
        appId: z.string(),
        defaultBranch: z.string().min(1),
      }),
    )
    .mutation(async ({ ctx, input }) => {
      const app = await db.query.apps
        .findFirst({
          where: (table, { and, eq }) =>
            and(eq(table.id, input.appId), eq(table.workspaceId, ctx.workspace.id)),
          columns: { id: true },
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
        .update(schema.apps)
        .set({ defaultBranch: input.defaultBranch, updatedAt: Date.now() })
        .where(eq(schema.apps.id, input.appId));

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
            eq(schema.githubRepoConnections.workspaceId, ctx.workspace.id),
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

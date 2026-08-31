import type { TRPCError } from "@trpc/server";
import { beforeEach, describe, expect, it, vi } from "vitest";

type Context = {
  workspace: { id: string; slug: string };
  user: { id: string };
  audit: { location: string; userAgent: string };
};

type RegisterInput = {
  state: string;
  installationId?: number;
  code?: string;
};

type RegisterResult =
  | { status: "authorization_required"; authorizationUrl: string }
  | {
      status: "registered";
      workspaceSlug: string;
      flow: "api" | "workspace" | "app";
      projectId: string | null;
      appId: string | null;
      returnTo: "settings" | null;
    };

type RegisterResolver = (opts: {
  ctx: Context;
  input: RegisterInput;
}) => Promise<RegisterResult>;

type PrepareWorkspaceResolver = (opts: { ctx: Context }) => Promise<{ state: string }>;

type InstallationRow = {
  pk: number;
  workspaceId: string;
  installationId: number;
};

type InstallationColumn = keyof InstallationRow;
type Predicate = { column: InstallationColumn; value: unknown } | { predicates: Predicate[] };

const state = vi.hoisted(() => ({
  prepareWorkspaceResolver: null as PrepareWorkspaceResolver | null,
  inputMutationResolvers: [] as RegisterResolver[],
  rows: [] as InstallationRow[],
  inserted: [] as Array<{ workspaceId: string; installationId: number }>,
  exchangedCodes: [] as string[],
  checkedInstallationIds: [] as number[],
  canAccessInstallation: true,
  exchangeError: null as Error | null,
}));

vi.mock("@/lib/db", () => {
  const installationTable = {
    pk: "pk",
    workspaceId: "workspaceId",
    installationId: "installationId",
  } as const;

  const eq = (column: InstallationColumn, value: unknown): Predicate => ({ column, value });
  const and = (...predicates: Predicate[]): Predicate => ({ predicates });
  const matches = (row: InstallationRow, predicate: Predicate): boolean =>
    "predicates" in predicate
      ? predicate.predicates.every((nested) => matches(row, nested))
      : row[predicate.column] === predicate.value;

  return {
    and,
    eq,
    schema: {
      githubAppInstallations: installationTable,
      githubRepoConnections: {},
      apps: {},
    },
    db: {
      query: {
        githubAppInstallations: {
          findFirst: async ({
            where,
          }: {
            where: (
              table: typeof installationTable,
              operators: { and: typeof and; eq: typeof eq },
            ) => Predicate;
          }) => state.rows.find((row) => matches(row, where(installationTable, { and, eq }))),
        },
        projects: {
          findFirst: async () => undefined,
        },
        apps: {
          findFirst: async () => undefined,
        },
      },
      transaction: async <T>(fn: (tx: unknown) => Promise<T>): Promise<T> => {
        const tx = {
          insert: () => ({
            values: (binding: { workspaceId: string; installationId: number }) => ({
              onDuplicateKeyUpdate: async () => {
                state.inserted.push({
                  workspaceId: binding.workspaceId,
                  installationId: binding.installationId,
                });
              },
            }),
          }),
        };
        return fn(tx);
      },
    },
  };
});

vi.mock("@/lib/audit", () => ({
  insertAuditLogs: async () => undefined,
}));

vi.mock("@/lib/env", () => ({
  githubAppEnv: () => ({
    GITHUB_APP_ID: 1,
    UNKEY_GITHUB_PRIVATE_KEY_PEM: "test-private-key",
  }),
  githubOAuthEnv: () => ({
    GITHUB_CLIENT_ID: "github-client-id",
    GITHUB_CLIENT_SECRET: "github-client-secret",
  }),
}));

vi.mock("@/lib/github", () => ({
  MAX_BRANCHES: 20,
  exchangeInstallationOAuthCode: async (code: string) => {
    state.exchangedCodes.push(code);
    if (state.exchangeError) {
      throw state.exchangeError;
    }
    return "user-token";
  },
  userCanAccessInstallation: async (_userToken: string, installationId: number) => {
    state.checkedInstallationIds.push(installationId);
    return state.canAccessInstallation;
  },
  getInstallationRepositories: async () => [],
  getMostActiveBranches: async () => [],
  getRepository: async () => null,
  getRepositoryBranches: async () => [],
  getRepositoryById: async () => null,
  getRepositoryTree: async () => ({ tree: [], truncated: false }),
  searchBranchesByPrefix: async () => [],
}));

vi.mock("../trpc", () => ({
  t: {
    router: <T>(routes: T): T => routes,
  },
  workspaceProcedure: {
    query: <T>(resolver: T): T => resolver,
    mutation: (resolver: PrepareWorkspaceResolver) => {
      state.prepareWorkspaceResolver = resolver;
      return resolver;
    },
    input: () => ({
      query: <T>(resolver: T): T => resolver,
      mutation: (resolver: RegisterResolver) => {
        state.inputMutationResolvers.push(resolver);
        return resolver;
      },
    }),
  },
}));

import "./github";

const context: Context = {
  workspace: { id: "ws_destination", slug: "destination" },
  user: { id: "user_1" },
  audit: { location: "127.0.0.1", userAgent: "vitest" },
};

function prepareWorkspaceState(): Promise<{ state: string }> {
  if (!state.prepareWorkspaceResolver) {
    throw new Error("prepareWorkspaceInstall resolver was not registered");
  }
  return state.prepareWorkspaceResolver({ ctx: context });
}

function registerInstallation(input: RegisterInput): Promise<RegisterResult> {
  const resolver = state.inputMutationResolvers[1];
  if (!resolver) {
    throw new Error("registerInstallation resolver was not registered");
  }
  return resolver({ ctx: context, input });
}

describe("registerInstallation", () => {
  beforeEach(() => {
    state.rows = [{ pk: 1, workspaceId: "ws_source", installationId: 42 }];
    state.inserted = [];
    state.exchangedCodes = [];
    state.checkedInstallationIds = [];
    state.canAccessInstallation = true;
    state.exchangeError = null;
  });

  it("binds one GitHub installation to another workspace after verifying the user", async () => {
    const signedState = await prepareWorkspaceState();

    await registerInstallation({
      state: signedState.state,
      installationId: 42,
      code: "oauth-code",
    });

    expect(state.exchangedCodes).toEqual(["oauth-code"]);
    expect(state.checkedInstallationIds).toEqual([42]);
    expect(state.inserted).toEqual([{ workspaceId: "ws_destination", installationId: 42 }]);
  });

  it("requests authorization when GitHub edits an existing installation without a code", async () => {
    const signedState = await prepareWorkspaceState();

    const authorization = await registerInstallation({
      state: signedState.state,
      installationId: 42,
    });
    if (authorization.status !== "authorization_required") {
      throw new Error("Expected GitHub authorization URL");
    }

    expect(state.inserted).toEqual([]);
    const authorizationUrl = new URL(authorization.authorizationUrl);
    expect(authorizationUrl.origin).toBe("https://github.com");
    expect(authorizationUrl.pathname).toBe("/login/oauth/authorize");
    expect(authorizationUrl.searchParams.get("client_id")).toBe("github-client-id");

    const refreshedState = authorizationUrl.searchParams.get("state");
    if (!refreshedState) {
      throw new Error("Authorization URL did not include state");
    }

    await registerInstallation({ state: refreshedState, code: "fresh-oauth-code" });

    expect(state.checkedInstallationIds).toEqual([42]);
    expect(state.inserted).toEqual([{ workspaceId: "ws_destination", installationId: 42 }]);
  });

  it("does not reauthorize an installation already linked to the workspace", async () => {
    state.rows = [{ pk: 1, workspaceId: "ws_destination", installationId: 42 }];
    const signedState = await prepareWorkspaceState();

    await registerInstallation({ state: signedState.state, installationId: 42 });

    expect(state.exchangedCodes).toEqual([]);
    expect(state.checkedInstallationIds).toEqual([]);
    expect(state.inserted).toEqual([{ workspaceId: "ws_destination", installationId: 42 }]);
  });

  it("rejects a new workspace binding when the GitHub user cannot access the installation", async () => {
    state.canAccessInstallation = false;
    const signedState = await prepareWorkspaceState();

    await expect(
      registerInstallation({
        state: signedState.state,
        installationId: 42,
        code: "oauth-code",
      }),
    ).rejects.toMatchObject({ code: "FORBIDDEN" } satisfies Partial<TRPCError>);
    expect(state.inserted).toEqual([]);
  });

  it("rejects a forged installation id in signed authorization state", async () => {
    const signedState = await prepareWorkspaceState();
    const authorization = await registerInstallation({
      state: signedState.state,
      installationId: 42,
    });
    if (authorization.status !== "authorization_required") {
      throw new Error("Expected GitHub authorization URL");
    }

    const authorizationUrl = new URL(authorization.authorizationUrl);
    const refreshedState = authorizationUrl.searchParams.get("state");
    if (!refreshedState) {
      throw new Error("Authorization URL did not include state");
    }
    const forgedState = refreshedState.replace('"installationId":42', '"installationId":99');
    if (forgedState === refreshedState) {
      throw new Error("Signed state did not include the installation id");
    }

    await expect(
      registerInstallation({ state: forgedState, code: "oauth-code" }),
    ).rejects.toMatchObject({ code: "BAD_REQUEST" } satisfies Partial<TRPCError>);
    expect(state.exchangedCodes).toEqual([]);
    expect(state.inserted).toEqual([]);
  });

  it("rejects an installation id swapped after authorization", async () => {
    const signedState = await prepareWorkspaceState();
    const authorization = await registerInstallation({
      state: signedState.state,
      installationId: 42,
    });
    if (authorization.status !== "authorization_required") {
      throw new Error("Expected GitHub authorization URL");
    }

    const authorizationUrl = new URL(authorization.authorizationUrl);
    const refreshedState = authorizationUrl.searchParams.get("state");
    if (!refreshedState) {
      throw new Error("Authorization URL did not include state");
    }

    await expect(
      registerInstallation({
        state: refreshedState,
        installationId: 99,
        code: "oauth-code",
      }),
    ).rejects.toMatchObject({ code: "BAD_REQUEST" } satisfies Partial<TRPCError>);
    expect(state.exchangedCodes).toEqual([]);
    expect(state.inserted).toEqual([]);
  });

  it("rejects a fake OAuth authorization code", async () => {
    state.exchangeError = new Error("bad verification code");
    const signedState = await prepareWorkspaceState();
    const consoleError = vi.spyOn(console, "error").mockImplementation(() => undefined);

    try {
      await expect(
        registerInstallation({
          state: signedState.state,
          installationId: 42,
          code: "fake-code",
        }),
      ).rejects.toMatchObject({ code: "BAD_REQUEST" } satisfies Partial<TRPCError>);
    } finally {
      consoleError.mockRestore();
    }
    expect(state.checkedInstallationIds).toEqual([]);
    expect(state.inserted).toEqual([]);
  });
});

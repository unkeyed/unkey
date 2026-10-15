import { newId } from "@unkey/id";
import { beforeEach, expect, it, vi } from "vitest";

type MutationOptions = {
  input: { deploymentId: string };
  ctx: {
    workspace: { id: string };
    user: { id: string };
    audit: { location: string; userAgent: string };
  };
};

const state = vi.hoisted(() => ({
  mutation: null as ((options: MutationOptions) => Promise<unknown>) | null,
  deployment: null as {
    id: string;
    projectId: string;
    project: { name: string };
  } | null,
  rows: [] as { payload: unknown }[],
}));

vi.mock("@/lib/trpc/trpc", () => ({
  ratelimit: { update: {} },
  withRatelimit: () => ({}),
  workspaceProcedure: {
    use: () => ({
      input: () => ({
        mutation: (mutation: (options: MutationOptions) => Promise<unknown>) => {
          state.mutation = mutation;
          return {};
        },
      }),
    }),
  },
}));

vi.mock("@/lib/db", () => ({
  db: {
    query: { deployments: { findFirst: async () => state.deployment } },
    insert: () => ({
      values: async (rows: { payload: unknown }[]) => {
        state.rows.push(...rows);
      },
    }),
  },
}));

vi.mock("@/lib/ctrl-client", () => ({
  createCtrlClient: () => ({ authorizeDeployment: async () => ({}) }),
}));

import "./authorize";

beforeEach(() => {
  state.rows = [];
  state.deployment = {
    id: newId("test"),
    projectId: newId("project"),
    project: { name: "Production API" },
  };
});

it("persists the authorized deployment target and retains the existing project target", async () => {
  if (!state.mutation || !state.deployment) {
    throw new Error("Authorization mutation and deployment must be initialized");
  }
  const workspaceId = newId("workspace");
  const actorId = newId("test");
  await state.mutation({
    input: { deploymentId: state.deployment.id },
    ctx: {
      workspace: { id: workspaceId },
      user: { id: actorId },
      audit: { location: "127.0.0.1", userAgent: "vitest" },
    },
  });

  expect(state.rows).toHaveLength(1);
  expect(state.rows[0].payload).toMatchObject({
    event: "deployment.authorize",
    workspace_id: workspaceId,
    actor: { type: "user", id: actorId },
    targets: [
      { type: "project", id: state.deployment.projectId, name: "Production API" },
      { type: "deployment", id: state.deployment.id },
    ],
  });
});

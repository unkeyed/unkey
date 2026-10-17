import { beforeEach, describe, expect, it, vi } from "vitest";

const rename = vi.fn();
const listEnvironmentVariables = vi.fn();
const setEnvironmentVariables = vi.fn();
const removeEnvironmentVariables = vi.fn();

vi.mock("@/lib/unkey-client", () => ({
  getUnkeyClient: () => ({
    environments: {
      listEnvironmentVariables,
      setEnvironmentVariables,
      removeEnvironmentVariables,
    },
  }),
  getErrorToast: () => ({ message: "" }),
}));
vi.mock("@unkey/ui", () => ({ toast: { promise: vi.fn() } }));
vi.mock("../client", async () => {
  const { QueryClient } = await import("@tanstack/query-core");
  return {
    queryClient: new QueryClient(),
    trpcClient: { deploy: { envVar: { rename: { mutate: rename } } } },
  };
});

const {
  PartialAddError,
  RenameAppliedError,
  WRITEONLY_VALUE_REQUIRED,
  addVariables,
  envVars,
  saveVariables,
  updateVariable,
} = await import("./env-vars");
const refetch = vi.spyOn(envVars.utils, "refetch").mockResolvedValue([]);

const variable = (key: string) => ({ key, value: "1", kind: "recoverable" as const });

const input = {
  projectId: "proj_1",
  appId: "app_1",
  environmentIds: ["env_prod", "env_prev"],
  variables: [variable("A"), { key: "B", value: "2", kind: "writeonly" as const }],
};

function existingKeys(byEnvironment: Record<string, string[]>) {
  listEnvironmentVariables.mockImplementation(async ({ environment }) => ({
    data: (byEnvironment[environment] ?? []).map((key) => ({
      key,
      kind: "recoverable",
      createdAt: 0,
    })),
  }));
}

describe("addVariables", () => {
  beforeEach(() => {
    refetch.mockClear();
    listEnvironmentVariables.mockReset();
    setEnvironmentVariables.mockReset();
    setEnvironmentVariables.mockResolvedValue({});
  });

  it("returns the taken keys, writes nothing and reloads once", async () => {
    existingKeys({ env_prev: ["B", "OTHER", "A"], env_prod: ["A"] });

    await expect(addVariables(input)).resolves.toEqual({ status: "taken", keys: ["A", "B"] });
    expect(listEnvironmentVariables.mock.calls.map(([req]) => req)).toEqual([
      { project: "proj_1", app: "app_1", environment: "env_prod", cursor: undefined },
      { project: "proj_1", app: "app_1", environment: "env_prev", cursor: undefined },
    ]);
    expect(setEnvironmentVariables).not.toHaveBeenCalled();
    expect(refetch).toHaveBeenCalledTimes(1);
  });

  it("writes each environment once, then reloads once", async () => {
    existingKeys({ env_prod: ["OTHER"] });

    await expect(addVariables(input)).resolves.toEqual({ status: "added", count: 2 });
    expect(setEnvironmentVariables.mock.calls.map(([req]) => req)).toEqual([
      {
        project: "proj_1",
        app: "app_1",
        environment: "env_prod",
        variables: [
          { key: "A", value: "1", kind: "recoverable" },
          { key: "B", value: "2", kind: "writeonly" },
        ],
      },
      {
        project: "proj_1",
        app: "app_1",
        environment: "env_prev",
        variables: [
          { key: "A", value: "1", kind: "recoverable" },
          { key: "B", value: "2", kind: "writeonly" },
        ],
      },
    ]);
    expect(refetch).toHaveBeenCalledTimes(1);
  });

  it("splits 120 variables into requests of 50, 50 and 20 per environment", async () => {
    existingKeys({});
    const variables = Array.from({ length: 120 }, (_, i) => variable(`K${i}`));

    await expect(addVariables({ ...input, variables })).resolves.toEqual({
      status: "added",
      count: 120,
    });
    const chunks = (environment: string) =>
      setEnvironmentVariables.mock.calls
        .map(([req]) => req)
        .filter((req) => req.environment === environment)
        .map((req) => [req.variables.length, req.variables[0].key]);
    expect(chunks("env_prod")).toEqual([
      [50, "K0"],
      [50, "K50"],
      [20, "K100"],
    ]);
    expect(chunks("env_prev")).toEqual([
      [50, "K0"],
      [50, "K50"],
      [20, "K100"],
    ]);
  });

  it("reloads after a failed write, then rejects with the cause", async () => {
    existingKeys({});
    setEnvironmentVariables.mockRejectedValue(new Error("rejected"));

    const err = await addVariables(input).catch((e: unknown) => e);
    expect(err).toEqual(new Error("rejected"));
    expect(err).not.toBeInstanceOf(PartialAddError);
    expect(refetch).toHaveBeenCalledTimes(1);
  });

  it("names the saved environments when only some writes land", async () => {
    existingKeys({});
    setEnvironmentVariables.mockImplementation(async ({ environment }) => {
      if (environment === "env_prev") {
        throw new Error("rejected");
      }
      return {};
    });

    const err = await addVariables(input).catch((e: unknown) => e);
    expect(err).toBeInstanceOf(PartialAddError);
    expect(err).toMatchObject({ savedEnvironmentIds: ["env_prod"], message: "rejected" });
    expect(refetch).toHaveBeenCalledTimes(1);
  });
});

describe("saveVariables", () => {
  const saved = (environmentId: string, key: string, value: string) => ({
    id: `${environmentId}:${key}`,
    key,
    value,
    type: "recoverable" as const,
    description: null,
    createdAt: 0,
    environmentId,
    projectId: "proj_1",
    appId: "app_1",
  });
  const changes = {
    edited: [saved("env_prod", "PORT", "3000"), saved("env_prev", "PORT", "3000")],
    removed: [saved("env_prod", "OLD", "x")],
  };

  beforeEach(() => {
    refetch.mockClear();
    listEnvironmentVariables.mockReset();
    setEnvironmentVariables.mockReset();
    setEnvironmentVariables.mockResolvedValue({});
    removeEnvironmentVariables.mockReset();
    removeEnvironmentVariables.mockResolvedValue({});
  });

  it("removes and edits saved keys per environment, then adds the new variables", async () => {
    existingKeys({});

    await expect(
      saveVariables({ ...input, variables: [variable("NEW")], ...changes }),
    ).resolves.toEqual({ status: "added", count: 1 });
    expect(removeEnvironmentVariables.mock.calls.map(([req]) => req)).toEqual([
      { project: "proj_1", app: "app_1", environment: "env_prod", variables: ["OLD"] },
    ]);
    const sets = (environment: string) =>
      setEnvironmentVariables.mock.calls
        .map(([req]) => req)
        .filter((req) => req.environment === environment)
        .map((req) => req.variables);
    const port = { key: "PORT", value: "3000", kind: "recoverable" };
    expect(sets("env_prod")).toEqual([[port], [variable("NEW")]]);
    expect(sets("env_prev")).toEqual([[port], [variable("NEW")]]);
  });

  it("rejects with the cause and adds nothing when an edit fails in only some environments", async () => {
    setEnvironmentVariables.mockImplementation(async ({ environment }) => {
      if (environment === "env_prev") {
        throw new Error("rejected");
      }
      return {};
    });

    const err = await saveVariables({ ...input, ...changes }).catch((e: unknown) => e);
    expect(err).toEqual(new Error("rejected"));
    expect(err).not.toBeInstanceOf(PartialAddError);
    expect(listEnvironmentVariables).not.toHaveBeenCalled();
  });

  it("rejects with the cause when no change lands", async () => {
    removeEnvironmentVariables.mockRejectedValue(new Error("down"));
    setEnvironmentVariables.mockRejectedValue(new Error("down"));

    const err = await saveVariables({ ...input, variables: [], ...changes }).catch(
      (e: unknown) => e,
    );
    expect(err).toEqual(new Error("down"));
    expect(err).not.toBeInstanceOf(PartialAddError);
  });
});

describe("updateVariable", () => {
  const original = {
    id: "env_prod:OLD",
    key: "OLD",
    value: "1",
    type: "recoverable" as const,
    description: null,
    createdAt: 0,
    environmentId: "env_prod",
    projectId: "proj_1",
    appId: "app_1",
  };

  it("renames first, then writes the new value", async () => {
    rename.mockResolvedValueOnce({});
    setEnvironmentVariables.mockResolvedValueOnce({});

    await updateVariable(original, { ...original, key: "NEW", value: "2" });

    expect(rename).toHaveBeenLastCalledWith({
      appId: "app_1",
      environmentIds: ["env_prod"],
      key: "OLD",
      newKey: "NEW",
    });
    expect(setEnvironmentVariables.mock.lastCall?.[0].variables).toEqual([
      { key: "NEW", value: "2", kind: "recoverable", description: undefined },
    ]);
    expect(rename.mock.invocationCallOrder[0]).toBeLessThan(
      setEnvironmentVariables.mock.invocationCallOrder[0],
    );
  });

  it("reports a kept rename when the value write after it fails", async () => {
    rename.mockResolvedValueOnce({});
    setEnvironmentVariables.mockRejectedValueOnce(new Error("gateway down"));

    const update = updateVariable(original, { ...original, key: "NEW", value: "2" });

    await expect(update).rejects.toBeInstanceOf(RenameAppliedError);
    await expect(update).rejects.toMatchObject({ newKey: "NEW", message: "gateway down" });
  });

  it("writes nothing when the rename fails", async () => {
    setEnvironmentVariables.mockClear();
    rename.mockRejectedValueOnce(new Error("taken"));

    await expect(updateVariable(original, { ...original, key: "NEW", value: "2" })).rejects.toThrow(
      "taken",
    );
    expect(setEnvironmentVariables).not.toHaveBeenCalled();
  });

  it("refuses a note change on a writeonly variable without a new value", async () => {
    rename.mockClear();
    setEnvironmentVariables.mockClear();
    const writeonly = { ...original, value: "", type: "writeonly" as const };

    await expect(
      updateVariable(writeonly, { ...writeonly, key: "NEW", description: "rotate monthly" }),
    ).rejects.toThrow(WRITEONLY_VALUE_REQUIRED);
    expect(rename).not.toHaveBeenCalled();
    expect(setEnvironmentVariables).not.toHaveBeenCalled();
  });

  it("renames a writeonly variable without writing a value", async () => {
    setEnvironmentVariables.mockClear();
    rename.mockResolvedValueOnce({});
    const writeonly = { ...original, value: "", type: "writeonly" as const };

    await updateVariable(writeonly, { ...writeonly, key: "NEW" });

    expect(setEnvironmentVariables).not.toHaveBeenCalled();
  });

  it("sends the known value when only the note of a recoverable variable changes", async () => {
    setEnvironmentVariables.mockResolvedValueOnce({});

    await updateVariable(original, { ...original, description: "rotate monthly" });

    expect(setEnvironmentVariables.mock.lastCall?.[0].variables).toEqual([
      { key: "OLD", value: "1", kind: "recoverable", description: "rotate monthly" },
    ]);
  });
});

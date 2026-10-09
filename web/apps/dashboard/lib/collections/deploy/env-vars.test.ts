import { describe, expect, it, vi } from "vitest";

const add = vi.fn();
const rename = vi.fn();
const setEnvironmentVariables = vi.fn();

vi.mock("@/lib/unkey-client", () => ({
  getUnkeyClient: () => ({ environments: { setEnvironmentVariables } }),
  getErrorToast: () => ({ message: "" }),
}));
vi.mock("@unkey/ui", () => ({ toast: { promise: vi.fn() } }));
vi.mock("../client", async () => {
  const { QueryClient } = await import("@tanstack/query-core");
  return {
    queryClient: new QueryClient(),
    trpcClient: { deploy: { envVar: { add: { mutate: add }, rename: { mutate: rename } } } },
  };
});

const { RenameAppliedError, addVariables, envVars, updateVariable } = await import("./env-vars");
const refetch = vi.spyOn(envVars.utils, "refetch").mockResolvedValue([]);

const input = {
  appId: "app_1",
  environmentIds: ["env_prod", "env_prev"],
  variables: [{ key: "A", value: "1", kind: "recoverable" as const }],
};

describe("addVariables", () => {
  it("returns the taken keys and reloads nothing", async () => {
    add.mockResolvedValueOnce({ status: "taken", keys: ["A"] });

    await expect(addVariables(input)).resolves.toEqual({ status: "taken", keys: ["A"] });
    expect(refetch).not.toHaveBeenCalled();
  });

  it("sends every environment in one request and reloads once", async () => {
    add.mockResolvedValueOnce({ status: "added", count: 1 });

    await expect(addVariables(input)).resolves.toEqual({ status: "added", count: 1 });
    expect(add).toHaveBeenLastCalledWith(input);
    expect(refetch).toHaveBeenCalledTimes(1);
  });

  it("reloads nothing when the add fails, since nothing was written", async () => {
    refetch.mockClear();
    add.mockRejectedValueOnce(new Error("rejected"));

    await expect(addVariables(input)).rejects.toThrow("rejected");
    expect(refetch).not.toHaveBeenCalled();
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
});

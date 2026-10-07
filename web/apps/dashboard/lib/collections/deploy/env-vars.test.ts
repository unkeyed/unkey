import { describe, expect, it, vi } from "vitest";

const stored = new Map<string, string[]>([["env_prod", ["TAKEN"]]]);
const rejected = new Set<string>();
const listEnvironmentVariables = vi.fn(async ({ environment }: { environment: string }) => ({
  data: (stored.get(environment) ?? []).map((key) => ({
    key,
    value: "",
    kind: "recoverable" as const,
    createdAt: 0,
  })),
}));
const setEnvironmentVariables = vi.fn(async ({ environment }: { environment: string }) => {
  if (rejected.has(environment)) {
    throw new Error("rejected");
  }
  return {};
});

vi.mock("@/lib/unkey-client", () => ({
  getUnkeyClient: () => ({
    environments: { listEnvironmentVariables, setEnvironmentVariables },
  }),
  getErrorToast: () => ({ message: "" }),
}));
vi.mock("@unkey/ui", () => ({ toast: { promise: vi.fn() } }));
vi.mock("../client", async () => {
  const { QueryClient } = await import("@tanstack/query-core");
  return { queryClient: new QueryClient() };
});

const { addVariables, envVars } = await import("./env-vars");
const refetch = vi.spyOn(envVars.utils, "refetch").mockResolvedValue([]);

const input = (environmentIds: string[], keys: string[]) => ({
  projectId: "proj_1",
  appId: "app_1",
  environmentIds,
  variables: keys.map((key) => ({ key, value: "1", kind: "recoverable" as const })),
});

describe("addVariables", () => {
  it("writes nothing and reloads nothing when a key is taken in one environment", async () => {
    const result = await addVariables(input(["env_prod", "env_prev"], ["A", "TAKEN"]));

    expect(result).toEqual({ status: "taken", keys: new Set(["TAKEN"]) });
    expect(setEnvironmentVariables).not.toHaveBeenCalled();
    expect(refetch).not.toHaveBeenCalled();
  });

  it("writes every environment, reloads once and counts distinct keys", async () => {
    const result = await addVariables(input(["env_prod", "env_prev"], ["A", "B", "C"]));

    expect(result).toEqual({ status: "added", count: 3 });
    expect(setEnvironmentVariables.mock.calls.map(([body]) => body.environment)).toEqual([
      "env_prod",
      "env_prev",
    ]);
    expect(refetch).toHaveBeenCalledTimes(1);
  });

  it("reloads after a write that failed in one environment", async () => {
    rejected.add("env_prev");
    refetch.mockClear();

    await expect(addVariables(input(["env_prod", "env_prev"], ["D"]))).rejects.toThrow("rejected");
    expect(refetch).toHaveBeenCalledTimes(1);
  });
});

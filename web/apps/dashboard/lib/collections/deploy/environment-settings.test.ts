import { describe, expect, it, vi } from "vitest";

const rejected = new Set<string>();
const updateSettings = vi.fn(async (body: { environment: string }) => {
  if (rejected.has(body.environment)) {
    throw new Error("rejected");
  }
  return {};
});
const listEnvironments = vi.fn(async () => ({
  data: [
    { id: "env_prod", slug: "production", kind: "production" as const },
    { id: "env_prev", slug: "preview", kind: "preview" as const },
  ],
}));
const toastPromise = vi.fn();

vi.mock("@/lib/unkey-client", () => ({
  getUnkeyClient: () => ({ environments: { listEnvironments, updateSettings } }),
  getErrorToast: () => ({ message: "", description: "" }),
}));
vi.mock("@unkey/ui", () => ({ toast: { promise: toastPromise } }));
vi.mock("../client", async () => {
  const { QueryClient } = await import("@tanstack/query-core");
  return { queryClient: new QueryClient() };
});

const { and, createLiveQueryCollection, eq } = await import("@tanstack/react-db");
const { environmentSettings } = await import("./environment-settings");

const scope = { project: "proj_1", app: "app_1", environment: "env_prod" };

describe("environmentSettings update", () => {
  it("leaves an auto-deploy save to its row and toasts a save that applies on the next deploy", async () => {
    const query = createLiveQueryCollection((q) =>
      q
        .from({ s: environmentSettings })
        .where(({ s }) => and(eq(s.projectId, "proj_1"), eq(s.appId, "app_1"))),
    );
    await query.preload();
    await vi.waitFor(() => expect(query.toArray).toHaveLength(2));

    const immediate = environmentSettings.update("env_prod", (d) => {
      d.autoDeploy = false;
    });
    await immediate.isPersisted.promise;
    expect(updateSettings).toHaveBeenLastCalledWith({ ...scope, autoDeploy: false });
    expect(toastPromise).not.toHaveBeenCalled();

    const loud = environmentSettings.update("env_prod", (d) => {
      d.port = 3000;
    });
    await loud.isPersisted.promise;
    expect(toastPromise).toHaveBeenCalledTimes(1);
  });

  it("reloads every environment and toasts once when one environment fails to save", async () => {
    rejected.add("env_prev");
    toastPromise.mockClear();
    const reads = listEnvironments.mock.calls.length;

    const tx = environmentSettings.update(["env_prod", "env_prev"], (drafts) =>
      drafts.forEach((d) => {
        d.port = 4000;
      }),
    );
    await expect(tx.isPersisted.promise).rejects.toThrow("rejected");

    expect(updateSettings).toHaveBeenCalledWith({ ...scope, port: 4000 });
    expect(updateSettings).toHaveBeenCalledWith({ ...scope, environment: "env_prev", port: 4000 });
    expect(toastPromise).toHaveBeenCalledTimes(1);
    await vi.waitFor(() => expect(listEnvironments.mock.calls.length).toBeGreaterThan(reads));
    rejected.clear();
  });
});

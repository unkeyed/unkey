import { describe, expect, it, vi } from "vitest";

const updateSettings = vi.fn(async () => ({}));
const listEnvironments = vi.fn(async () => ({
  data: [{ id: "env_prod", slug: "production", kind: "production" as const }],
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
const { appliesOnNextDeploy, environmentSettings } = await import("./environment-settings");

const scope = { project: "proj_1", app: "app_1", environment: "env_prod" };

describe("appliesOnNextDeploy", () => {
  it("is false when only auto-deploy changed", () => {
    expect(appliesOnNextDeploy({ ...scope, autoDeploy: false })).toBe(false);
  });

  it("is true when a runtime field changed", () => {
    expect(appliesOnNextDeploy({ ...scope, autoDeploy: true, port: 3000 })).toBe(true);
  });
});

describe("environmentSettings update", () => {
  it("skips the collection toast for a silent mutation and shows it otherwise", async () => {
    const query = createLiveQueryCollection((q) =>
      q
        .from({ s: environmentSettings })
        .where(({ s }) => and(eq(s.projectId, "proj_1"), eq(s.appId, "app_1"))),
    );
    await query.preload();
    await vi.waitFor(() => expect(query.toArray).toHaveLength(1));

    const silent = environmentSettings.update("env_prod", { metadata: { silent: true } }, (d) => {
      d.autoDeploy = false;
    });
    await silent.isPersisted.promise;
    expect(updateSettings).toHaveBeenLastCalledWith({ ...scope, autoDeploy: false });
    expect(toastPromise).not.toHaveBeenCalled();

    const loud = environmentSettings.update("env_prod", (d) => {
      d.port = 3000;
    });
    await loud.isPersisted.promise;
    expect(toastPromise).toHaveBeenCalledTimes(1);
  });
});

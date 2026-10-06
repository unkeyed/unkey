import { describe, expect, it, vi } from "vitest";

const updateSettings = vi.fn(async (body: Record<string, unknown>) => ({ data: body }));

vi.mock("@/lib/unkey-client", () => ({
  getUnkeyClient: () => ({ environments: { updateSettings } }),
  getErrorToast: () => "",
}));
vi.mock("../client", async () => {
  const { QueryClient } = await import("@tanstack/query-core");
  return { queryClient: new QueryClient() };
});

const { applyDefaultSettings, defaultSettingsRow } = await import("./environment-settings");

describe("defaultSettingsRow", () => {
  it("is the row a fresh environment has after the defaults are written", async () => {
    const initial = { regionNames: ["us-east-1", "eu-west-1"], port: 3000 };
    const row = defaultSettingsRow("proj_1", "app_1", "env_1", initial);
    expect(row).toEqual({
      environmentId: "env_1",
      projectId: "proj_1",
      appId: "app_1",
      autoDeploy: true,
      dockerfile: "",
      dockerContext: ".",
      buildCommand: "",
      watchPaths: [],
      port: 3000,
      cpuMillicores: 250,
      memoryMib: 256,
      storageMib: 0,
      command: [],
      healthcheck: null,
      regions: [
        { name: "us-east-1", replicasMin: 1, replicasMax: 1 },
        { name: "eu-west-1", replicasMin: 1, replicasMax: 1 },
      ],
      shutdownSignal: "SIGTERM",
      upstreamProtocol: "http1",
      openapiSpecPath: null,
    });

    await applyDefaultSettings("proj_1", "app_1", "env_1", initial);
    expect(updateSettings).toHaveBeenCalledWith(
      expect.objectContaining({
        environment: "env_1",
        rootDirectory: ".",
        port: 3000,
        vCpus: 0.25,
        memoryMib: 256,
        regions: [
          { name: "us-east-1", replicas: { min: 1, max: 1 } },
          { name: "eu-west-1", replicas: { min: 1, max: 1 } },
        ],
      }),
    );
  });
});

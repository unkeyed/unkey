import type { EnvironmentSettings } from "@/lib/collections/deploy/environment-settings";
import { describe, expect, it } from "vitest";
import {
  commandSectionSchema,
  computeSectionSchema,
  readRuntime,
  runtimeSchema,
  writeRuntime,
} from "./model";

const stored: EnvironmentSettings = {
  environmentId: "env_prod",
  projectId: "proj_1",
  appId: "app_1",
  autoDeploy: true,
  dockerfile: "",
  dockerContext: ".",
  buildCommand: "",
  watchPaths: [],
  port: 8080,
  cpuMillicores: 250,
  memoryMib: 256,
  storageMib: 0,
  command: ["sh", "-c", "node a.js && node b.js"],
  healthcheck: null,
  regions: [{ name: "us-east-1", replicasMin: 1, replicasMax: 1 }],
  shutdownSignal: "SIGTERM",
  upstreamProtocol: "http1",
  openapiSpecPath: null,
};

function save(edit: (values: ReturnType<typeof readRuntime>) => void): EnvironmentSettings {
  const saved = readRuntime(stored);
  const values = structuredClone(saved);
  edit(values);
  const draft = structuredClone(stored);
  writeRuntime(draft, values, saved);
  return draft;
}

describe("writeRuntime", () => {
  it("keeps a stored command with spaced arguments when only compute changes", () => {
    const draft = save((values) => {
      values.compute.replicas = { min: 1, max: 3 };
    });
    expect(draft.command).toEqual(["sh", "-c", "node a.js && node b.js"]);
    expect(draft.regions).toEqual([{ name: "us-east-1", replicasMin: 1, replicasMax: 3 }]);
  });

  it("splits an edited command on whitespace", () => {
    const draft = save((values) => {
      values.command = "  node   server.js --port 3000 ";
    });
    expect(draft.command).toEqual(["node", "server.js", "--port", "3000"]);
  });

  it("leaves the health check off when only the port changes", () => {
    const draft = save((values) => {
      values.port = 3000;
    });
    expect(draft.port).toBe(3000);
    expect(draft.healthcheck).toBeNull();
  });
});

describe("runtimeSchema", () => {
  const values = readRuntime(stored);

  it("accepts an empty interval while the health check path is empty", () => {
    const result = runtimeSchema.safeParse({
      ...values,
      healthcheck: { method: "GET", path: "", intervalSeconds: Number.NaN },
    });
    expect(result.success).toBe(true);
  });

  it("rejects an empty interval once a health check path is set", () => {
    const result = runtimeSchema.safeParse({
      ...values,
      healthcheck: { method: "GET", path: "/health", intervalSeconds: Number.NaN },
    });
    expect(result.error?.issues.map((issue) => [issue.path, issue.message])).toEqual([
      [["healthcheck", "intervalSeconds"], "Use 1 to 3600 seconds"],
    ]);
  });
});

describe("section schemas", () => {
  it("lets the start command save while the stored regions are empty", () => {
    const values = {
      ...readRuntime(stored),
      compute: { ...readRuntime(stored).compute, regions: [] },
    };
    expect(runtimeSchema.safeParse(values).success).toBe(false);
    expect(commandSectionSchema.safeParse(values).success).toBe(true);
  });

  it("lets compute save while the stored port is out of range", () => {
    const values = { ...readRuntime(stored), port: 0 };
    expect(computeSectionSchema.safeParse(values).success).toBe(true);
  });
});

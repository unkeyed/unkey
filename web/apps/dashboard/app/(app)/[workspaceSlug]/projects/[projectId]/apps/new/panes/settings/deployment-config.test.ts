import type { EnvironmentSettings } from "@/lib/collections/deploy/environment-settings";
import { describe, expect, it } from "vitest";
import {
  applyDeploymentConfig,
  deploymentConfigSchema,
  findDockerfiles,
  readDeploymentConfig,
  suggestRootDirectories,
} from "./deployment-config";

const settings: EnvironmentSettings = {
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
  command: [],
  healthcheck: null,
  regions: [
    { name: "us-east-1", replicasMin: 2, replicasMax: 4 },
    { name: "eu-central-1", replicasMin: 1, replicasMax: 1 },
  ],
  shutdownSignal: "SIGTERM",
  upstreamProtocol: "http1",
  openapiSpecPath: null,
};

describe("readDeploymentConfig", () => {
  it("reads the fields the step shows", () => {
    expect(readDeploymentConfig(settings)).toEqual({
      dockerContext: ".",
      regions: ["us-east-1", "eu-central-1"],
      port: 8080,
      dockerfile: "",
      buildCommand: "",
      startCommand: "",
      size: { cpuMillicores: 250, memoryMib: 256 },
    });
  });
});

describe("applyDeploymentConfig", () => {
  it("writes the step fields and keeps replicas of kept regions", () => {
    const draft = structuredClone(settings);
    applyDeploymentConfig(draft, {
      dockerContext: "services/api",
      regions: ["eu-central-1", "ap-southeast-1"],
      port: 3000,
      dockerfile: "Dockerfile",
      buildCommand: "pnpm turbo build --filter=api",
      startCommand: "  pnpm --filter api   start ",
      size: { cpuMillicores: 1000, memoryMib: 2048 },
    });

    expect(draft).toEqual({
      ...settings,
      dockerContext: "services/api",
      port: 3000,
      dockerfile: "Dockerfile",
      buildCommand: "pnpm turbo build --filter=api",
      command: ["pnpm", "--filter", "api", "start"],
      cpuMillicores: 1000,
      memoryMib: 2048,
      regions: [
        { name: "eu-central-1", replicasMin: 1, replicasMax: 1 },
        { name: "ap-southeast-1", replicasMin: 2, replicasMax: 4 },
      ],
    });
  });

  it("gives new regions one replica when the environment has none", () => {
    const draft = structuredClone({ ...settings, regions: [] });
    applyDeploymentConfig(draft, { ...readDeploymentConfig(settings), regions: ["us-west-2"] });

    expect(draft.regions).toEqual([{ name: "us-west-2", replicasMin: 1, replicasMax: 1 }]);
  });
});

describe("startCommand", () => {
  it("round-trips the stored command", () => {
    const config = readDeploymentConfig({ ...settings, command: ["node", "dist/server.js"] });
    expect(config.startCommand).toBe("node dist/server.js");
  });

  it("stores an empty command as automatic", () => {
    const draft = structuredClone({ ...settings, command: ["node", "server.js"] });
    applyDeploymentConfig(draft, { ...readDeploymentConfig(settings), startCommand: "   " });
    expect(draft.command).toEqual([]);
  });
});

describe("deploymentConfigSchema", () => {
  const valid = readDeploymentConfig(settings);

  it.each([".", "api", "services/api"])("accepts root directory %s", (dockerContext) => {
    expect(deploymentConfigSchema.safeParse({ ...valid, dockerContext }).success).toBe(true);
  });

  it.each(["", "/api", "./api", "api/../web", "api\\web"])(
    "rejects root directory %s",
    (dockerContext) => {
      expect(deploymentConfigSchema.safeParse({ ...valid, dockerContext }).success).toBe(false);
    },
  );

  it.each([0, 65536, 80.5, Number.NaN])("rejects port %s", (port) => {
    expect(deploymentConfigSchema.safeParse({ ...valid, port }).success).toBe(false);
  });

  it("rejects an empty region list", () => {
    expect(deploymentConfigSchema.safeParse({ ...valid, regions: [] }).success).toBe(false);
  });
});

const tree = [
  { path: "package.json", type: "blob" },
  { path: "apps", type: "tree" },
  { path: "apps/web/package.json", type: "blob" },
  { path: "apps/web/Dockerfile", type: "blob" },
  { path: "services/api/go.mod", type: "blob" },
  { path: "services/api/main.go", type: "blob" },
  { path: "Dockerfile.dev", type: "blob" },
];

describe("suggestRootDirectories", () => {
  it("lists the root first, then directories that hold a project marker", () => {
    expect(suggestRootDirectories(tree)).toEqual([
      { path: ".", marker: "Repository root" },
      { path: "apps/web", marker: "package.json" },
      { path: "services/api", marker: "go.mod" },
    ]);
  });
});

describe("findDockerfiles", () => {
  it("lists Dockerfiles relative to the root directory", () => {
    expect(findDockerfiles(tree, ".")).toEqual(["apps/web/Dockerfile", "Dockerfile.dev"]);
    expect(findDockerfiles(tree, "apps/web")).toEqual(["Dockerfile"]);
    expect(findDockerfiles(tree, "services/api")).toEqual([]);
  });
});

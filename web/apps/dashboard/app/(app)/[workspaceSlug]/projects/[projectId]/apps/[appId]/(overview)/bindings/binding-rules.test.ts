import { describe, expect, it } from "vitest";
import {
  type Binding,
  type Deployment,
  type Environment,
  describeTarget,
  ruleOf,
} from "./binding-rules";

const environments: Environment[] = [
  { id: "caller-prod", appId: "caller", slug: "production", kind: "production" },
  { id: "caller-preview", appId: "caller", slug: "preview", kind: "preview" },
  { id: "caller-canary", appId: "caller", slug: "canary", kind: "production" },
  { id: "db-prod", appId: "db", slug: "production", kind: "production" },
  { id: "db-canary", appId: "db", slug: "canary", kind: "production" },
];
const deployments: Deployment[] = [
  { id: "db-pinned", appId: "db", environmentId: "db-canary", status: "ready" },
  { id: "db-stopped", appId: "db", environmentId: "db-canary", status: "stopped" },
];
const [callerProd, callerPreview, callerCanary] = environments;

function binding(overrides: Partial<Binding>): Binding {
  return {
    id: "bind_1",
    appId: "caller",
    environmentId: "caller-prod",
    targetAppId: "db",
    targetType: "automatic",
    targetEnvironmentId: null,
    targetDeploymentId: null,
    ...overrides,
  };
}

describe("describeTarget", () => {
  const common = { targetName: "db", environments, deployments };

  it("says which target version each automatic rule connects to", () => {
    expect(
      describeTarget({ ...common, binding: binding({}), callerEnvironment: callerProd }),
    ).toEqual({
      text: "Follows db’s live production deployment, including rollbacks.",
      unavailable: false,
    });
    expect(
      describeTarget({ ...common, binding: binding({}), callerEnvironment: callerPreview }).text,
    ).toBe("Connects to db’s preview from the same Git branch, if there is one.");
  });

  it("describes explicit environments and pins by what they connect to", () => {
    expect(
      describeTarget({
        ...common,
        binding: binding({ targetType: "environment", targetEnvironmentId: "db-canary" }),
        callerEnvironment: callerProd,
      }).text,
    ).toBe("Connects to db’s latest canary version.");
    expect(
      describeTarget({
        ...common,
        binding: binding({ targetType: "environment", targetEnvironmentId: "db-prod" }),
        callerEnvironment: callerCanary,
      }).text,
    ).toBe("Follows db’s live production deployment, including rollbacks.");
    expect(
      describeTarget({
        ...common,
        binding: binding({ targetType: "deployment", targetDeploymentId: "db-pinned" }),
        callerEnvironment: callerProd,
      }).text,
    ).toBe("Always connects to db deployment db-pinned.");
  });

  it("marks a deleted target environment unavailable instead of falling back", () => {
    const result = describeTarget({
      ...common,
      binding: binding({ targetType: "environment", targetEnvironmentId: "deleted" }),
      callerEnvironment: callerProd,
    });
    expect(result.unavailable).toBe(true);
    expect(result.text).not.toContain("production");
  });

  it("marks a stopped pinned deployment unavailable", () => {
    expect(
      describeTarget({
        ...common,
        binding: binding({ targetType: "deployment", targetDeploymentId: "db-stopped" }),
        callerEnvironment: callerProd,
      }).unavailable,
    ).toBe(true);
  });

  it("never mentions ports or protocols", () => {
    for (const rule of [
      binding({}),
      binding({ targetType: "environment", targetEnvironmentId: "db-canary" }),
      binding({ targetType: "deployment", targetDeploymentId: "db-pinned" }),
    ]) {
      expect(
        describeTarget({ ...common, binding: rule, callerEnvironment: callerProd }).text,
      ).not.toMatch(/port|tcp|udp|http/i);
    }
  });
});

describe("ruleOf", () => {
  it("rejects a discriminator without its target column", () => {
    expect(ruleOf(binding({ targetType: "environment" }))).toBeUndefined();
    expect(ruleOf(binding({ targetType: "deployment", targetDeploymentId: "d" }))).toEqual({
      targetType: "deployment",
      targetDeploymentId: "d",
    });
  });
});

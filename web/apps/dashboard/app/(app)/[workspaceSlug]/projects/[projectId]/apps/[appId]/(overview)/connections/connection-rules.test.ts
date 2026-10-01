import { describe, expect, it } from "vitest";
import {
  type Connection,
  type Deployment,
  type Environment,
  describeTarget,
  ruleOf,
} from "./connection-rules";

const environments: Environment[] = [
  { id: "caller-prod", appId: "caller", slug: "production", kind: "production" },
  { id: "caller-preview", appId: "caller", slug: "preview", kind: "preview" },
  { id: "caller-canary", appId: "caller", slug: "canary", kind: "production" },
  { id: "db-prod", appId: "db", slug: "production", kind: "production" },
  { id: "db-canary", appId: "db", slug: "canary", kind: "production" },
  { id: "db-preview", appId: "db", slug: "test", kind: "preview" },
];
const deployments: Deployment[] = [
  { id: "db-pinned", appId: "db", environmentId: "db-canary", status: "ready" },
  { id: "db-stopped", appId: "db", environmentId: "db-canary", status: "stopped" },
];
const [callerProd, callerPreview, callerCanary] = environments;

function connection(overrides: Partial<Connection>): Connection {
  return {
    id: "conn_1",
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
      describeTarget({ ...common, connection: connection({}), callerEnvironment: callerProd }),
    ).toEqual({
      text: "Follows db’s live production deployment, including rollbacks.",
      unavailable: false,
    });
    expect(
      describeTarget({ ...common, connection: connection({}), callerEnvironment: callerPreview })
        .text,
    ).toBe("Connects to db’s preview from the same Git branch, if there is one.");
    expect(
      describeTarget({ ...common, connection: connection({}), callerEnvironment: callerCanary })
        .text,
    ).toBe("Follows db’s live production deployment, including rollbacks.");
  });

  it("describes explicit environments and pins by what they connect to", () => {
    expect(
      describeTarget({
        ...common,
        connection: connection({ targetType: "environment", targetEnvironmentId: "db-canary" }),
        callerEnvironment: callerProd,
      }).text,
    ).toBe("Follows db’s live production deployment, including rollbacks.");
    expect(
      describeTarget({
        ...common,
        connection: connection({ targetType: "environment", targetEnvironmentId: "db-preview" }),
        callerEnvironment: callerProd,
      }).text,
    ).toBe("Connects to db’s latest test version.");
    expect(
      describeTarget({
        ...common,
        connection: connection({ targetType: "environment", targetEnvironmentId: "db-prod" }),
        callerEnvironment: callerCanary,
      }).text,
    ).toBe("Follows db’s live production deployment, including rollbacks.");
    expect(
      describeTarget({
        ...common,
        connection: connection({ targetType: "deployment", targetDeploymentId: "db-pinned" }),
        callerEnvironment: callerProd,
      }).text,
    ).toBe("Always connects to db deployment db-pinned.");
  });

  it("marks a deleted target environment unavailable instead of falling back", () => {
    const result = describeTarget({
      ...common,
      connection: connection({ targetType: "environment", targetEnvironmentId: "deleted" }),
      callerEnvironment: callerProd,
    });
    expect(result.unavailable).toBe(true);
    expect(result.text).not.toContain("production");
  });

  it("marks a stopped pinned deployment unavailable", () => {
    expect(
      describeTarget({
        ...common,
        connection: connection({ targetType: "deployment", targetDeploymentId: "db-stopped" }),
        callerEnvironment: callerProd,
      }).unavailable,
    ).toBe(true);
  });

  it("never mentions ports or protocols", () => {
    for (const rule of [
      connection({}),
      connection({ targetType: "environment", targetEnvironmentId: "db-canary" }),
      connection({ targetType: "deployment", targetDeploymentId: "db-pinned" }),
    ]) {
      expect(
        describeTarget({ ...common, connection: rule, callerEnvironment: callerProd }).text,
      ).not.toMatch(/port|tcp|udp|http/i);
    }
  });
});

describe("ruleOf", () => {
  it("rejects a discriminator without its target column", () => {
    expect(ruleOf(connection({ targetType: "environment" }))).toBeUndefined();
    expect(ruleOf(connection({ targetType: "deployment", targetDeploymentId: "d" }))).toEqual({
      targetType: "deployment",
      targetDeploymentId: "d",
    });
  });
});

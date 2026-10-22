import { newId } from "@unkey/id";
import { describe, expect, it } from "vitest";
import {
  type Connection,
  type Deployment,
  type Environment,
  describeTarget,
  ruleOf,
} from "./connection-rules";

const caller = newId("app");
const db = newId("app");
const callerProdId = newId("environment");
const dbProd = newId("environment");
const dbCanary = newId("environment");
const dbPreview = newId("environment");
const dbPinned = newId("test");
const dbStopped = newId("test");
const environments: Environment[] = [
  { id: callerProdId, appId: caller, slug: "production", kind: "production" },
  { id: newId("environment"), appId: caller, slug: "preview", kind: "preview" },
  { id: newId("environment"), appId: caller, slug: "canary", kind: "production" },
  { id: dbProd, appId: db, slug: "production", kind: "production" },
  { id: dbCanary, appId: db, slug: "canary", kind: "production" },
  { id: dbPreview, appId: db, slug: "test", kind: "preview" },
];
const deployments: Deployment[] = [
  { id: dbPinned, appId: db, environmentId: dbCanary, status: "ready" },
  { id: dbStopped, appId: db, environmentId: dbCanary, status: "stopped" },
];
const [callerProd, callerPreview, callerCanary] = environments;

function connection(overrides: Partial<Connection>): Connection {
  return {
    id: newId("connection"),
    appId: caller,
    environmentId: callerProdId,
    targetAppId: db,
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
        connection: connection({ targetType: "environment", targetEnvironmentId: dbCanary }),
        callerEnvironment: callerProd,
      }).text,
    ).toBe("Follows db’s live production deployment, including rollbacks.");
    expect(
      describeTarget({
        ...common,
        connection: connection({ targetType: "environment", targetEnvironmentId: dbPreview }),
        callerEnvironment: callerProd,
      }).text,
    ).toBe("Connects to db’s latest test version.");
    expect(
      describeTarget({
        ...common,
        connection: connection({ targetType: "environment", targetEnvironmentId: dbProd }),
        callerEnvironment: callerCanary,
      }).text,
    ).toBe("Follows db’s live production deployment, including rollbacks.");
    expect(
      describeTarget({
        ...common,
        connection: connection({ targetType: "deployment", targetDeploymentId: dbPinned }),
        callerEnvironment: callerProd,
      }).text,
    ).toBe(`Always connects to db deployment ${dbPinned}.`);
  });

  it("marks a deleted target environment unavailable instead of falling back", () => {
    const result = describeTarget({
      ...common,
      connection: connection({
        targetType: "environment",
        targetEnvironmentId: newId("environment"),
      }),
      callerEnvironment: callerProd,
    });
    expect(result.unavailable).toBe(true);
    expect(result.text).not.toContain("production");
  });

  it("marks a stopped pinned deployment unavailable", () => {
    expect(
      describeTarget({
        ...common,
        connection: connection({ targetType: "deployment", targetDeploymentId: dbStopped }),
        callerEnvironment: callerProd,
      }).unavailable,
    ).toBe(true);
  });

  it("does not describe a pin missing from the choices page as stopped", () => {
    const result = describeTarget({
      ...common,
      connection: connection({ targetType: "deployment", targetDeploymentId: newId("test") }),
      callerEnvironment: callerProd,
    });
    expect(result).toEqual({
      text: "The pinned db deployment is unavailable. Choose another target.",
      unavailable: true,
    });
  });

  it("does not describe a failed pinned deployment as stopped", () => {
    const result = describeTarget({
      ...common,
      deployments: common.deployments.map((deployment) => ({ ...deployment, status: "failed" })),
      connection: connection({ targetType: "deployment", targetDeploymentId: dbPinned }),
      callerEnvironment: callerProd,
    });
    expect(result).toEqual({
      text: "The pinned db deployment is unavailable. Choose another target.",
      unavailable: true,
    });
  });

  it("never mentions ports or protocols", () => {
    for (const rule of [
      connection({}),
      connection({ targetType: "environment", targetEnvironmentId: dbCanary }),
      connection({ targetType: "deployment", targetDeploymentId: dbPinned }),
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
    const deploymentId = newId("test");
    expect(
      ruleOf(connection({ targetType: "deployment", targetDeploymentId: deploymentId })),
    ).toEqual({
      targetType: "deployment",
      targetDeploymentId: deploymentId,
    });
  });
});

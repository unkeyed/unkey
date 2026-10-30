import { newId } from "@unkey/id";
import { describe, expect, it } from "vitest";
import {
  type Deployment,
  type Environment,
  type TargetContext,
  type TargetRule,
  describeTarget,
  ruleOf,
  sameRule,
  targetError,
  targetLabel,
} from "./target";

const caller = newId("app");
const db = newId("app");
const dbProd = newId("environment");
const dbCanary = newId("environment");
const dbPreview = newId("environment");
const dbPinned = newId("test");
const dbStopped = newId("test");
const environments: Environment[] = [
  { id: newId("environment"), appId: caller, slug: "production", kind: "production" },
  { id: newId("environment"), appId: caller, slug: "preview", kind: "preview" },
  { id: dbProd, appId: db, slug: "production", kind: "production" },
  { id: dbCanary, appId: db, slug: "canary", kind: "production" },
  { id: dbPreview, appId: db, slug: "test", kind: "preview" },
];
const deployments: Deployment[] = [
  { id: dbPinned, appId: db, environmentId: dbCanary, status: "ready" },
  { id: dbStopped, appId: db, environmentId: dbCanary, status: "stopped" },
];
const fromProduction: TargetContext = {
  targetName: "db",
  environments,
  deployments,
};

const automatic: TargetRule = { targetType: "automatic" };
const environment = (id: string): TargetRule => ({
  targetType: "environment",
  targetEnvironmentId: id,
});
const pinned = (id: string): TargetRule => ({ targetType: "deployment", targetDeploymentId: id });

describe("describeTarget", () => {
  it("follows the target's live production deployment by default", () => {
    expect(describeTarget(automatic, fromProduction)).toEqual({
      text: "Follows db’s live production deployment, including rollbacks.",
      unavailable: false,
    });
  });

  it("describes explicit environments and pins by what they connect to", () => {
    expect(describeTarget(environment(dbCanary), fromProduction).text).toBe(
      "Follows db’s live production deployment, including rollbacks.",
    );
    expect(describeTarget(environment(dbPreview), fromProduction).text).toBe(
      "Connects to db’s latest test version.",
    );
    expect(describeTarget(pinned(dbPinned), fromProduction).text).toBe(
      `Always connects to db deployment ${dbPinned}.`,
    );
  });

  it("marks a deleted target environment unavailable instead of falling back", () => {
    expect(describeTarget(environment(newId("environment")), fromProduction)).toEqual({
      text: "The selected db environment was deleted. Choose another target.",
      unavailable: true,
    });
  });

  it("marks a stopped pinned deployment unavailable", () => {
    expect(describeTarget(pinned(dbStopped), fromProduction)).toEqual({
      text: "The pinned db deployment is stopped. Choose another target.",
      unavailable: true,
    });
  });

  it("does not describe a pin missing from the choices page as stopped", () => {
    expect(describeTarget(pinned(newId("test")), fromProduction)).toEqual({
      text: "The pinned db deployment is unavailable. Choose another target.",
      unavailable: true,
    });
  });

  it("does not describe a failed pinned deployment as stopped", () => {
    const failed = deployments.map((deployment) => ({ ...deployment, status: "failed" as const }));
    expect(describeTarget(pinned(dbPinned), { ...fromProduction, deployments: failed })).toEqual({
      text: "The pinned db deployment is unavailable. Choose another target.",
      unavailable: true,
    });
  });

  it("never mentions ports or protocols", () => {
    for (const rule of [automatic, environment(dbCanary), pinned(dbPinned)]) {
      expect(describeTarget(rule, fromProduction).text).not.toMatch(/port|tcp|udp|http/i);
    }
  });
});

describe("ruleOf", () => {
  it("reads each target type from the API columns", () => {
    expect(ruleOf({ targetType: "automatic" })).toEqual(automatic);
    expect(ruleOf({ targetType: "environment", targetEnvironmentId: dbCanary })).toEqual(
      environment(dbCanary),
    );
    expect(ruleOf({ targetType: "deployment", targetDeploymentId: dbPinned })).toEqual(
      pinned(dbPinned),
    );
  });

  it("reads a row without its target id as an unpicked target", () => {
    expect(ruleOf({ targetType: "environment" })).toEqual(environment(""));
    expect(ruleOf({ targetType: "deployment" })).toEqual(pinned(""));
  });
});

describe("targetError", () => {
  it("asks for the missing id of each target type", () => {
    expect(targetError(automatic)).toBeNull();
    expect(targetError(environment(dbCanary))).toBeNull();
    expect(targetError(environment(""))).toBe("Choose an environment");
    expect(targetError(pinned(""))).toBe("Choose a deployment");
  });
});

describe("sameRule", () => {
  it("compares the target type and id", () => {
    expect(sameRule(environment(dbCanary), environment(dbCanary))).toBe(true);
    expect(sameRule(environment(dbCanary), environment(dbProd))).toBe(false);
    expect(sameRule(automatic, environment(""))).toBe(false);
  });
});

describe("targetLabel", () => {
  it("names the target each rule resolves to", () => {
    expect(targetLabel(automatic, fromProduction)).toBe("Production");
    expect(targetLabel(environment(dbCanary), fromProduction)).toBe("canary");
    expect(targetLabel(pinned(dbPinned), fromProduction)).toBe(dbPinned);
  });

  it("calls out a deleted environment or a pin without an id", () => {
    expect(targetLabel(environment(newId("environment")), fromProduction)).toBe(
      "Deleted environment",
    );
    expect(targetLabel(pinned(""), fromProduction)).toBe("Unavailable deployment");
  });
});

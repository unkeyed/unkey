import { newId } from "@unkey/id";
import { describe, expect, it } from "vitest";
import type { Environment, TargetDeployment, TargetRule } from "./target";
import {
  type TargetChoices,
  deploymentOptions,
  mergeDeploymentTargets,
  selectValueOf,
  targetHint,
  targetOptions,
} from "./target-options";

const db = newId("app");
const dbProd: Environment = {
  id: newId("environment"),
  appId: db,
  slug: "production",
  kind: "production",
};
const dbCanary: Environment = {
  id: newId("environment"),
  appId: db,
  slug: "canary",
  kind: "production",
};
const dbPreview: Environment = {
  id: newId("environment"),
  appId: db,
  slug: "preview",
  kind: "preview",
};
const other: Environment = {
  id: newId("environment"),
  appId: newId("app"),
  slug: "production",
  kind: "production",
};

function deployment(overrides: Partial<TargetDeployment> = {}): TargetDeployment {
  return {
    id: newId("test"),
    appId: db,
    environmentId: dbCanary.id,
    status: "ready",
    gitBranch: "main",
    image: null,
    ...overrides,
  };
}

const ready = deployment();
const choices: TargetChoices = {
  targetAppId: db,
  targetName: "db",
  environments: [dbProd, dbCanary, dbPreview, other],
  deployments: [ready],
};
const automatic: TargetRule = { targetType: "automatic" };

describe("connection target deployment pagination", () => {
  it("keeps a live pin resolved independently when it is beyond the choices page", () => {
    const pageOfChoices = Array.from({ length: 100 }, () => ({ id: newId("test") }));
    const livePin = { id: newId("test"), status: "ready" };
    const another = { id: newId("test"), status: "ready" };

    expect(mergeDeploymentTargets([livePin], pageOfChoices)).toEqual([livePin, ...pageOfChoices]);
    expect(
      mergeDeploymentTargets([livePin], [{ id: livePin.id, status: "stopped" }, another]),
    ).toEqual([livePin, another]);
  });
});

describe("select values", () => {
  it("gives every option a value that maps back to its own rule", () => {
    const pin: TargetRule = { targetType: "deployment", targetDeploymentId: ready.id };
    for (const option of targetOptions(pin, choices)) {
      expect(selectValueOf(option.rule)).toBe(option.value);
    }
  });

  it("keeps the picked deployment on the pin option", () => {
    const pin: TargetRule = { targetType: "deployment", targetDeploymentId: ready.id };
    expect(targetOptions(pin, choices).find((option) => option.type === "pin")?.rule).toEqual(pin);
  });
});

describe("targetOptions", () => {
  const shape = (current: TargetRule, input: TargetChoices = choices) =>
    targetOptions(current, input).map((option) => [option.type, option.value]);

  it("offers automatic, the target app's other environments and a pin", () => {
    expect(shape(automatic)).toEqual([
      ["automatic", "automatic"],
      ["environment", `env:${dbCanary.id}`],
      ["environment", `env:${dbPreview.id}`],
      ["pin", "deployment"],
    ]);
  });

  it("keeps production listed when production already targets it", () => {
    const current: TargetRule = { targetType: "environment", targetEnvironmentId: dbProd.id };
    expect(shape(current)).toContainEqual(["environment", `env:${dbProd.id}`]);
  });

  it("lists a deleted environment so the select still shows the saved value", () => {
    const gone = newId("environment");
    const current: TargetRule = { targetType: "environment", targetEnvironmentId: gone };
    expect(shape(current)).toContainEqual(["deleted", `env:${gone}`]);
  });
});

describe("deploymentOptions", () => {
  it("lists a saved pin that is not on the loaded pages first", () => {
    const saved = newId("test");
    expect(deploymentOptions(saved, [ready]).map((option) => [option.type, option.value])).toEqual([
      ["missing", saved],
      ["deployment", ready.id],
    ]);
    expect(deploymentOptions("", [ready]).map((option) => option.type)).toEqual(["deployment"]);
  });
});

describe("targetHint", () => {
  it("shows the field error first", () => {
    expect(targetHint(automatic, "Choose an environment", choices)).toEqual({
      tone: "error",
      text: "Choose an environment",
    });
  });

  it("warns about an unavailable target and stays quiet otherwise", () => {
    const stopped = deployment({ status: "stopped" });
    const pin: TargetRule = { targetType: "deployment", targetDeploymentId: stopped.id };
    expect(targetHint(pin, undefined, { ...choices, deployments: [stopped] })).toEqual({
      tone: "warning",
      text: "The pinned db deployment is stopped. Choose another target.",
    });
    expect(targetHint(automatic, undefined, choices)).toBeNull();
    expect(
      targetHint({ targetType: "deployment", targetDeploymentId: "" }, undefined, choices),
    ).toBeNull();
  });
});

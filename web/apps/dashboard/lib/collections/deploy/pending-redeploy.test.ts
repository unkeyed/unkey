import { describe, expect, it } from "vitest";
import { appliesOnNextDeploy } from "./pending-redeploy";

const scope = { project: "proj_1", app: "app_1", environment: "env_prod" };

describe("appliesOnNextDeploy", () => {
  it("is false when only auto-deploy changed", () => {
    expect(appliesOnNextDeploy({ ...scope, autoDeploy: false })).toBe(false);
  });

  it("is true when a runtime field changed", () => {
    expect(appliesOnNextDeploy({ ...scope, autoDeploy: true, port: 3000 })).toBe(true);
  });
});

import { newId } from "@unkey/id";
import { describe, expect, it } from "vitest";
import type { CallerEnvironment, Callers } from "./callers";
import type { ConnectionOp } from "./plan";
import type { TargetCaller, TargetRule } from "./target";
import { whoGetsLines } from "./who-gets";

const production: CallerEnvironment = { id: newId("environment"), kind: "production" };
const preview: CallerEnvironment = { id: newId("environment"), kind: "preview" };
const edgeApi = newId("app");
const edgeProduction = newId("environment");

const targets = {
  environments: [
    { id: edgeProduction, appId: edgeApi, slug: "production", kind: "production" as const },
  ],
  deployments: [],
};
const callers: Callers<TargetCaller> = {
  production: { environment: production, targets },
  preview: { environment: preview, targets },
};

const upsert = (environment: CallerEnvironment, rule: TargetRule): ConnectionOp => ({
  type: "update",
  environment,
  id: newId("connection"),
  rule,
});
const automatic: TargetRule = { targetType: "automatic" };
const PRODUCTION_LINE = { subject: "Production", rest: "picks it up after you redeploy." };

describe("whoGetsLines", () => {
  it("describes the preview line by its target rule", () => {
    expect(
      whoGetsLines(
        [upsert(production, automatic), upsert(preview, automatic)],
        callers,
        "edge-api",
      ),
    ).toEqual([
      PRODUCTION_LINE,
      {
        subject: "New preview deployments",
        rest: "get it when edge-api has a preview on the same branch. Existing ones keep their setup until they redeploy.",
      },
    ]);
    expect(
      whoGetsLines(
        [upsert(preview, { targetType: "environment", targetEnvironmentId: edgeProduction })],
        callers,
        "edge-api",
      ),
    ).toEqual([
      {
        subject: "New preview deployments",
        rest: "connect to edge-api production. Existing ones keep their setup until they redeploy.",
      },
    ]);
    expect(
      whoGetsLines(
        [upsert(preview, { targetType: "deployment", targetDeploymentId: newId("fake") })],
        callers,
        "edge-api",
      ),
    ).toEqual([
      {
        subject: "New preview deployments",
        rest: "connect to the pinned edge-api deployment. Existing ones keep their setup until they redeploy.",
      },
    ]);
  });

  it("says previews stop getting it when the preview companion is deleted", () => {
    expect(
      whoGetsLines(
        [{ type: "delete", environment: preview, id: newId("connection") }],
        callers,
        "edge-api",
      ),
    ).toEqual([{ subject: "Preview deployments", rest: "stop getting it after they redeploy." }]);
  });
});

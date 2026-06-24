import { describe, expect, test } from "vitest";
import { getDeploymentActionEligibility } from "./deployment-action-eligibility";

describe("getDeploymentActionEligibility", () => {
  const baseCtx = {
    selectedDeployment: { id: "dep-1", status: "ready" as const },
    currentDeploymentId: "dep-current",
    isRolledBack: false,
    environmentSlug: "production",
  };

  const cases = [
    {
      name: "ready + production + not current + not rolled back → rollback=true, promote=true, redeploy=true",
      ctx: baseCtx,
      expected: {
        canRollback: true,
        canPromote: true,
        canRedeploy: true,
        canCancel: false,
        canStop: false,
        canWake: false,
      },
    },
    {
      name: "ready + production + not current + rolled back → rollback=true, promote=true, redeploy=true",
      ctx: { ...baseCtx, isRolledBack: true },
      expected: {
        canRollback: true,
        canPromote: true,
        canRedeploy: true,
        canCancel: false,
        canStop: false,
        canWake: false,
      },
    },
    {
      name: "ready + production + is current + not rolled back → promote=false, rollback=false, redeploy=true",
      ctx: {
        ...baseCtx,
        selectedDeployment: { id: "dep-current", status: "ready" as const },
      },
      expected: {
        canRollback: false,
        canPromote: false,
        canRedeploy: true,
        canCancel: false,
        canStop: false,
        canWake: false,
      },
    },
    {
      name: "ready + production + is current + rolled back → rollback=false, promote=true, redeploy=true",
      ctx: {
        ...baseCtx,
        selectedDeployment: { id: "dep-current", status: "ready" as const },
        isRolledBack: true,
      },
      expected: {
        canRollback: false,
        canPromote: true,
        canRedeploy: true,
        canCancel: false,
        canStop: false,
        canWake: false,
      },
    },
    {
      name: "ready + staging → redeploy and stop",
      ctx: { ...baseCtx, environmentSlug: "staging" },
      expected: {
        canRollback: false,
        canPromote: false,
        canRedeploy: true,
        canCancel: false,
        canStop: true,
        canWake: false,
      },
    },
    {
      name: "failed → only redeploy",
      ctx: {
        ...baseCtx,
        selectedDeployment: { id: "dep-1", status: "failed" as const },
      },
      expected: {
        canRollback: false,
        canPromote: false,
        canRedeploy: true,
        canCancel: false,
        canStop: false,
        canWake: false,
      },
    },
    {
      name: "pending → all false",
      ctx: {
        ...baseCtx,
        selectedDeployment: { id: "dep-1", status: "pending" as const },
      },
      expected: {
        canRollback: false,
        canPromote: false,
        canRedeploy: false,
        canCancel: true,
        canStop: false,
        canWake: false,
      },
    },
    {
      name: "building → all false",
      ctx: {
        ...baseCtx,
        selectedDeployment: { id: "dep-1", status: "building" as const },
      },
      expected: {
        canRollback: false,
        canPromote: false,
        canRedeploy: false,
        canCancel: true,
        canStop: false,
        canWake: false,
      },
    },
    {
      name: "ready + production + no current deployment → all false except redeploy",
      ctx: { ...baseCtx, currentDeploymentId: null },
      expected: {
        canRollback: false,
        canPromote: false,
        canRedeploy: true,
        canCancel: false,
        canStop: false,
        canWake: false,
      },
    },
    {
      name: "null environment slug → only redeploy if ready",
      ctx: { ...baseCtx, environmentSlug: null },
      expected: {
        canRollback: false,
        canPromote: false,
        canRedeploy: true,
        canCancel: false,
        canStop: false,
        canWake: false,
      },
    },
    {
      name: "stopped + preview → wake and redeploy",
      ctx: {
        ...baseCtx,
        selectedDeployment: { id: "dep-1", status: "stopped" as const },
        environmentSlug: "preview",
      },
      expected: {
        canRollback: false,
        canPromote: false,
        canRedeploy: true,
        canCancel: false,
        canStop: false,
        canWake: true,
      },
    },
    {
      name: "stopped + production → redeploy only",
      ctx: {
        ...baseCtx,
        selectedDeployment: { id: "dep-1", status: "stopped" as const },
      },
      expected: {
        canRollback: false,
        canPromote: false,
        canRedeploy: true,
        canCancel: false,
        canStop: false,
        canWake: false,
      },
    },
  ] as const;

  for (const { name, ctx, expected } of cases) {
    test(name, () => {
      expect(getDeploymentActionEligibility(ctx)).toEqual(expected);
    });
  }
});

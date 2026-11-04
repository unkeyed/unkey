import type { EnvironmentKind } from "@/lib/deploy-data";
import { match } from "@unkey/match";
import type { Callers } from "./callers";
import type { ConnectionOp } from "./plan";
import {
  type TargetCaller,
  type TargetContext,
  type TargetRule,
  targetContextOf,
  targetLabel,
} from "./target";

export type WhoGetsLine = { subject: string; rest: string };

const KEEP_SETUP = "Existing ones keep their setup until they redeploy.";

function previewReach(rule: TargetRule, context: TargetContext): string {
  const { targetName } = context;
  return match(rule)
    .with(
      { targetType: "automatic" },
      () => `get it when ${targetName} has a preview on the same branch.`,
    )
    .with(
      { targetType: "environment" },
      () => `connect to ${targetName} ${targetLabel(rule, context)}.`,
    )
    .with({ targetType: "deployment" }, () => `connect to the pinned ${targetName} deployment.`)
    .exhaustive();
}

const WHO_GETS: Record<
  EnvironmentKind,
  { reaches: (rule: TargetRule, context: TargetContext) => WhoGetsLine; stops: WhoGetsLine[] }
> = {
  production: {
    reaches: () => ({ subject: "Production", rest: "picks it up after you redeploy." }),
    stops: [],
  },
  preview: {
    reaches: (rule, context) => ({
      subject: "New preview deployments",
      rest: `${previewReach(rule, context)} ${KEEP_SETUP}`,
    }),
    stops: [{ subject: "Preview deployments", rest: "stop getting it after they redeploy." }],
  },
};

export function whoGetsLines(
  ops: readonly ConnectionOp[],
  callers: Callers<TargetCaller>,
  targetName: string,
): WhoGetsLine[] {
  return ops.flatMap((op) => {
    const { kind } = op.environment;
    if (op.type === "delete") {
      return WHO_GETS[kind].stops;
    }
    const caller = callers[kind];
    return caller ? [WHO_GETS[kind].reaches(op.rule, targetContextOf(caller, targetName))] : [];
  });
}

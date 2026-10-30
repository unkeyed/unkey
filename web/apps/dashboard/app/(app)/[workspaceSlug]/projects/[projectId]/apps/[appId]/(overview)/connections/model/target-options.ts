import { match } from "@unkey/match";
import {
  AUTOMATIC_TARGET_LABEL,
  type Environment,
  type TargetContext,
  type TargetDeployment,
  type TargetRule,
  describeTarget,
  isProduction,
  targetError,
} from "./target";

export type TargetChoices = TargetContext<TargetDeployment> & { targetAppId: string };

type OptionShape =
  | { type: "automatic"; label: string }
  | { type: "environment"; environment: Environment }
  | { type: "deleted" }
  | { type: "pin" };

export type TargetOption = OptionShape & { value: string; rule: TargetRule };

export type DeploymentOption =
  | { type: "missing"; value: string }
  | { type: "deployment"; value: string; deployment: TargetDeployment };

export type TargetHint = { tone: "warning" | "error"; text: string };

const PIN_VALUE = "deployment";

export function mergeDeploymentTargets<T extends { id: string }>(pins: T[], choices: T[]): T[] {
  const seen = new Set<string>();
  return [...pins, ...choices].filter((deployment) => {
    if (seen.has(deployment.id)) {
      return false;
    }
    seen.add(deployment.id);
    return true;
  });
}

export function selectValueOf(rule: TargetRule): string {
  return match(rule)
    .with({ targetType: "automatic" }, () => "automatic")
    .with({ targetType: "environment" }, (r) => `env:${r.targetEnvironmentId}`)
    .with({ targetType: "deployment" }, () => PIN_VALUE)
    .exhaustive();
}

function option(shape: OptionShape, rule: TargetRule): TargetOption {
  return { ...shape, rule, value: selectValueOf(rule) };
}

export function targetOptions(current: TargetRule, choices: TargetChoices): TargetOption[] {
  const targetEnvs = choices.environments.filter((env) => env.appId === choices.targetAppId);
  const coveredByAutomatic = targetEnvs.find(isProduction);
  const currentEnvId = current.targetType === "environment" ? current.targetEnvironmentId : "";
  const deleted: TargetOption[] =
    currentEnvId !== "" && !targetEnvs.some((env) => env.id === currentEnvId)
      ? [option({ type: "deleted" }, current)]
      : [];
  return [
    option({ type: "automatic", label: AUTOMATIC_TARGET_LABEL }, { targetType: "automatic" }),
    ...targetEnvs
      .filter((env) => env !== coveredByAutomatic || env.id === currentEnvId)
      .map((environment) =>
        option(
          { type: "environment", environment },
          { targetType: "environment", targetEnvironmentId: environment.id },
        ),
      ),
    ...deleted,
    option(
      { type: "pin" },
      current.targetType === "deployment"
        ? current
        : { targetType: "deployment", targetDeploymentId: "" },
    ),
  ];
}

export function deploymentOptions(
  current: string,
  deployments: TargetDeployment[],
): DeploymentOption[] {
  const missing: DeploymentOption[] =
    current !== "" && !deployments.some((deployment) => deployment.id === current)
      ? [{ type: "missing", value: current }]
      : [];
  return [
    ...missing,
    ...deployments.map((deployment) => ({
      type: "deployment" as const,
      value: deployment.id,
      deployment,
    })),
  ];
}

export function targetHint(
  target: TargetRule,
  error: string | undefined,
  choices: TargetChoices,
): TargetHint | null {
  if (error) {
    return { tone: "error", text: error };
  }
  if (targetError(target)) {
    return null;
  }
  const { text, unavailable } = describeTarget(target, choices);
  return unavailable ? { tone: "warning", text } : null;
}

"use client";

import {
  EnvironmentIcon,
  EnvironmentLabel,
  environmentName,
} from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/_components/environment-label";
import { Switch } from "@/components/ui/switch";
import { ENVIRONMENT_KINDS, type EnvironmentKind } from "@/lib/collections/deploy/environments";
import { IconChevronRightOutline12 } from "@unkey/icons";
import { match } from "@unkey/match";
import { Button, FormSelect, type FormSelectOption } from "@unkey/ui";
import { cn } from "cn";
import { useState } from "react";
import { useFormContext, useFormState, useWatch } from "react-hook-form";
import {
  AUTOMATIC_TARGET_LABEL,
  type ConnectionTargets,
  type Environment,
  columnsOf,
  describeTarget,
  isProduction,
  mergeDeploymentTargets,
} from "../../connection-rules";
import type { TargetDeployments } from "../../hooks/use-connections-data";
import {
  type ConnectionFormValues,
  type EnvDraft,
  type TargetDraft,
  ruleOfDraft,
  targetError,
} from "../../model/form";
import type { ConnectionEnvs } from "../../model/merge";

type TargetChoices = {
  targetAppId: string;
  targetAppName: string;
  environments: Environment[];
  deployments: ConnectionTargets["deployments"];
  page: TargetDeployments;
  requiresExplicitPreviewTarget: boolean;
};

export function EnvironmentsCard({
  envs,
  targetAppId,
  targetAppName,
  targets,
  page,
  requiresExplicitPreviewTarget,
}: {
  envs: ConnectionEnvs;
  targetAppId: string;
  targetAppName: string;
  targets: ConnectionTargets;
  page: TargetDeployments;
  requiresExplicitPreviewTarget: boolean;
}) {
  const { control } = useFormContext<ConnectionFormValues>();
  const { errors } = useFormState({ control, name: "envs" });
  const drafts = useWatch({ control, name: "envs" });
  const [advancedOpen, setAdvancedOpen] = useState(() =>
    ENVIRONMENT_KINDS.some((kind) => drafts[kind].target.mode !== "automatic"),
  );
  const enabledKinds = ENVIRONMENT_KINDS.filter((kind) => drafts[kind].enabled);
  const choices: TargetChoices = {
    targetAppId,
    targetAppName,
    environments: targets.environments,
    deployments: mergeDeploymentTargets(
      targets.deployments.filter((deployment) => deployment.appId === targetAppId),
      page.deployments,
    ),
    page,
    requiresExplicitPreviewTarget,
  };

  return (
    <div className="flex flex-col gap-1.5">
      <span className="text-sm font-medium text-gray-12">Environments</span>
      <div className="flex flex-col divide-y divide-grayA-4 rounded-lg border border-grayA-4">
        {ENVIRONMENT_KINDS.map((kind) => (
          <EnvSwitchRow key={kind} kind={kind} callerEnv={envs[kind]} choices={choices} />
        ))}
        {enabledKinds.length > 0 ? (
          <div>
            <button
              type="button"
              aria-expanded={advancedOpen}
              onClick={() => setAdvancedOpen((open) => !open)}
              className="flex w-full items-center gap-1.5 px-3 py-2 text-xs text-gray-11 hover:text-gray-12"
            >
              <IconChevronRightOutline12
                className={cn("size-3 transition-transform", advancedOpen && "rotate-90")}
              />
              Advanced
            </button>
            {advancedOpen ? (
              <div className="flex flex-col gap-3 px-3 pb-3">
                {enabledKinds.map((kind) => (
                  <EnvTargetPicker
                    key={kind}
                    kind={kind}
                    callerEnv={envs[kind]}
                    choices={choices}
                  />
                ))}
                <p className="text-xs text-gray-10">
                  Pick another environment, or pin one deployment.
                </p>
              </div>
            ) : null}
          </div>
        ) : null}
      </div>
      {errors.envs?.message ? (
        <p role="alert" className="text-xs text-error-11">
          {errors.envs.message}
        </p>
      ) : null}
    </div>
  );
}

const AUTOMATIC = "automatic";
const DEPLOYMENT = "deployment";

function selectValueOf(target: TargetDraft): string {
  return match(target)
    .with({ mode: "environment" }, ({ environmentId }) => environmentId)
    .with({ mode: "automatic" }, () => AUTOMATIC)
    .with({ mode: "deployment" }, () => DEPLOYMENT)
    .exhaustive();
}

function targetFromValue(value: string, previous: TargetDraft): TargetDraft {
  if (value === AUTOMATIC) {
    return { mode: "automatic" };
  }
  if (value === DEPLOYMENT) {
    return previous.mode === "deployment" ? previous : { mode: "deployment", deploymentId: "" };
  }
  return { mode: "environment", environmentId: value };
}

function targetOptions(
  kind: EnvironmentKind,
  current: TargetDraft,
  choices: TargetChoices,
): FormSelectOption[] {
  const targetEnvs = choices.environments.filter((env) => env.appId === choices.targetAppId);
  const coveredByAutomatic = kind === "production" ? targetEnvs.find(isProduction) : undefined;
  const currentEnvId = current.mode === "environment" ? current.environmentId : "";
  const automaticAllowed = !(kind === "preview" && choices.requiresExplicitPreviewTarget);
  const deleted =
    currentEnvId !== "" && !targetEnvs.some((env) => env.id === currentEnvId)
      ? [{ value: currentEnvId, label: "Deleted environment", disabled: true }]
      : [];
  return [
    ...(automaticAllowed || current.mode === "automatic"
      ? [{ value: AUTOMATIC, label: AUTOMATIC_TARGET_LABEL[kind] }]
      : []),
    ...targetEnvs
      .filter((env) => env !== coveredByAutomatic || env.id === currentEnvId)
      .map((env) => ({
        value: env.id,
        label: <EnvironmentLabel environment={env} className="text-sm text-gray-12" />,
      })),
    ...deleted,
    { value: DEPLOYMENT, label: "Pin a deployment…" },
  ];
}

function deploymentOptions(current: string, choices: TargetChoices): FormSelectOption[] {
  const missing =
    current !== "" && !choices.deployments.some((deployment) => deployment.id === current)
      ? [{ value: current, label: "Unavailable deployment", disabled: true }]
      : [];
  return [
    ...missing,
    ...choices.deployments.map((deployment) => ({
      value: deployment.id,
      disabled: deployment.status !== "ready",
      label: (
        <span className="flex min-w-0 items-center gap-2">
          <span className="truncate font-mono text-xs">{deployment.id}</span>
          <span className="truncate text-xs text-gray-9">
            {deployment.gitBranch ?? deployment.image ?? ""}
            {deployment.status === "ready" ? "" : ` (${deployment.status})`}
          </span>
        </span>
      ),
    })),
  ];
}

type Hint = { tone: "warning" | "error"; text: string };

const HINT_CLASS: Record<Hint["tone"], string> = {
  warning: "text-warning-11",
  error: "text-error-11",
};

function hintOf(
  kind: EnvironmentKind,
  target: TargetDraft,
  error: string | undefined,
  choices: TargetChoices,
): Hint | null {
  if (error) {
    return { tone: "error", text: error };
  }
  if (targetError(target)) {
    return null;
  }
  const { text, unavailable } = describeTarget({
    connection: columnsOf(ruleOfDraft(target)),
    targetName: choices.targetAppName,
    callerEnvironment: { kind },
    environments: choices.environments,
    deployments: choices.deployments,
  });
  return unavailable ? { tone: "warning", text } : null;
}

function summaryOf(
  kind: EnvironmentKind,
  target: TargetDraft,
  choices: TargetChoices,
): string | null {
  if (kind !== "preview" || target.mode !== "automatic") {
    return null;
  }
  return `Previews reach ${choices.targetAppName} on the same branch, if it has one.`;
}

function useEnvDraft(kind: EnvironmentKind) {
  const { control, setValue } = useFormContext<ConnectionFormValues>();
  const draft = useWatch({ control, name: `envs.${kind}` });
  const update = (next: EnvDraft) => setValue(`envs.${kind}`, next, { shouldDirty: true });
  return { draft, update };
}

function EnvSwitchRow({
  kind,
  callerEnv,
  choices,
}: {
  kind: EnvironmentKind;
  callerEnv: Environment | null;
  choices: TargetChoices;
}) {
  const { draft, update } = useEnvDraft(kind);
  const environment = callerEnv ?? { slug: kind, kind };
  const summary = summaryOf(kind, draft.target, choices);

  return (
    <div className="flex items-center justify-between gap-3 px-3 py-2.5">
      <div className="flex min-w-0 flex-col gap-0.5">
        <span
          className={cn(
            "flex items-center gap-1.5 text-sm font-medium",
            draft.enabled ? "text-gray-12" : "text-gray-10",
          )}
        >
          <EnvironmentIcon kind={kind} className="text-gray-11" />
          {environmentName(environment.slug)}
        </span>
        {draft.enabled && summary ? <span className="text-xs text-gray-11">{summary}</span> : null}
      </div>
      <Switch
        size="sm"
        checked={draft.enabled}
        disabled={!callerEnv}
        aria-label={`Connect from ${environment.slug}`}
        onCheckedChange={(enabled) => update({ ...draft, enabled })}
      />
    </div>
  );
}

function EnvTargetPicker({
  kind,
  callerEnv,
  choices,
}: {
  kind: EnvironmentKind;
  callerEnv: Environment | null;
  choices: TargetChoices;
}) {
  const { control } = useFormContext<ConnectionFormValues>();
  const { draft, update } = useEnvDraft(kind);
  const { errors } = useFormState({ control, name: `envs.${kind}.target` });
  const environment = callerEnv ?? { slug: kind, kind };
  const hint = hintOf(kind, draft.target, errors.envs?.[kind]?.target?.message, choices);
  const deploymentId = draft.target.mode === "deployment" ? draft.target.deploymentId : "";

  return (
    <div className="grid grid-cols-[5.5rem_minmax(0,1fr)] items-center gap-2">
      <span className="text-xs text-gray-11">{environmentName(environment.slug)}</span>
      <FormSelect
        aria-label={`Target in ${environment.slug}`}
        size="sm"
        placeholder="Choose a target"
        options={targetOptions(kind, draft.target, choices)}
        value={selectValueOf(draft.target)}
        onValueChange={(value) =>
          update({ ...draft, target: targetFromValue(value, draft.target) })
        }
      />
      {draft.target.mode === "deployment" ? (
        <div className="col-start-2 flex items-center gap-2">
          <FormSelect
            aria-label={`Deployment in ${environment.slug}`}
            size="sm"
            className="min-w-0 flex-1"
            placeholder="Choose a deployment"
            options={deploymentOptions(deploymentId, choices)}
            value={deploymentId}
            onValueChange={(next) =>
              update({ ...draft, target: { mode: "deployment", deploymentId: next } })
            }
          />
          {choices.page.hasMore ? (
            <Button
              type="button"
              variant="outline"
              size="sm"
              loading={choices.page.loadingMore}
              onClick={choices.page.loadMore}
            >
              Load more
            </Button>
          ) : null}
        </div>
      ) : null}
      {hint ? (
        <p className={cn("col-start-2 text-xs", HINT_CLASS[hint.tone])}>{hint.text}</p>
      ) : null}
    </div>
  );
}

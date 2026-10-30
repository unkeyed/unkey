"use client";

import { EnvironmentLabel } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/_components/environment-label";
import type { ListView } from "@/lib/loadable/list";
import { getErrorMessage } from "@/lib/unkey-client";
import { match } from "@unkey/match";
import { Button, FormSelect, type FormSelectOption } from "@unkey/ui";
import type { ReactNode } from "react";
import { useFormContext, useFormState, useWatch } from "react-hook-form";
import type { ConnectionFormValues } from "../../model/form";
import type { TargetDeployment } from "../../model/target";
import {
  type DeploymentOption,
  type TargetChoices,
  type TargetHint,
  type TargetOption,
  deploymentOptions,
  selectValueOf,
  targetHint,
  targetOptions,
} from "../../model/target-options";

export function TargetField({
  choices,
  deployments,
}: {
  choices: TargetChoices;
  deployments: ListView<TargetDeployment>;
}) {
  const { control, setValue } = useFormContext<ConnectionFormValues>();
  const target = useWatch({ control, name: "target" });
  const { errors } = useFormState({ control, name: "target" });
  const hint = targetHint(target, errors.target?.message, choices);
  const hintProps = hint ? HINT_PROPS[hint.tone](hint.text) : { description: TARGET_DESCRIPTION };
  const options = targetOptions(target, choices);
  const update = (next: ConnectionFormValues["target"]) =>
    setValue("target", next, { shouldDirty: true, shouldValidate: errors.target !== undefined });

  return (
    <div className="flex flex-col gap-2">
      <FormSelect
        label="Target"
        size="sm"
        placeholder="Choose a target"
        {...hintProps}
        options={options.map(targetSelectOption)}
        value={selectValueOf(target)}
        onValueChange={(value) => {
          const picked = options.find((candidate) => candidate.value === value);
          if (picked) {
            update(picked.rule);
          }
        }}
      />
      {target.targetType === "deployment" ? (
        <DeploymentPicker
          value={target.targetDeploymentId}
          choices={choices.deployments}
          deployments={deployments}
          onValueChange={(id) => update({ targetType: "deployment", targetDeploymentId: id })}
        />
      ) : null}
    </div>
  );
}

const TARGET_DESCRIPTION = "Pick another environment, or pin one deployment.";

const HINT_PROPS: Record<
  TargetHint["tone"],
  (text: string) => { description: ReactNode; error?: string }
> = {
  warning: (text) => ({ description: <span className="text-warning-11">{text}</span> }),
  error: (text) => ({ description: TARGET_DESCRIPTION, error: text }),
};

function targetSelectOption(option: TargetOption): FormSelectOption {
  return match(option)
    .with({ type: "automatic" }, ({ value, label }) => ({ value, label }))
    .with({ type: "environment" }, ({ value, environment }) => ({
      value,
      label: <EnvironmentLabel environment={environment} className="text-sm text-gray-12" />,
    }))
    .with({ type: "deleted" }, ({ value }) => ({
      value,
      label: "Deleted environment",
      disabled: true,
    }))
    .with({ type: "pin" }, ({ value }) => ({ value, label: "Pin a deployment…" }))
    .exhaustive();
}

function deploymentSelectOption(option: DeploymentOption): FormSelectOption {
  return match(option)
    .with({ type: "missing" }, ({ value }) => ({
      value,
      label: "Unavailable deployment",
      disabled: true,
    }))
    .with({ type: "deployment" }, ({ value, deployment }) => ({
      value,
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
    }))
    .exhaustive();
}

type PagerState = { action: ReactNode; error: string | null };

function pagerOf(deployments: ListView<TargetDeployment>): PagerState | null {
  const retry = (error: unknown, onRetry: () => void, retrying: boolean): PagerState => ({
    error: getErrorMessage(error, "We couldn't load deployments."),
    action: (
      <Button type="button" variant="outline" size="sm" loading={retrying} onClick={onRetry}>
        Retry
      </Button>
    ),
  });
  return match(deployments)
    .with({ status: "failed" }, ({ error, retry: onRetry }) => retry(error, onRetry, false))
    .with({ status: "ready" }, ({ footer }) =>
      match(footer)
        .with({ kind: "more" }, ({ onLoadMore }) => ({
          error: null,
          action: (
            <Button type="button" variant="outline" size="sm" onClick={onLoadMore}>
              Load more
            </Button>
          ),
        }))
        .with({ kind: "loadingMore" }, () => ({
          error: null,
          action: (
            <Button type="button" variant="outline" size="sm" loading>
              Load more
            </Button>
          ),
        }))
        .with({ kind: "error" }, ({ error, onRetry, retrying }) => retry(error, onRetry, retrying))
        .with({ kind: "hidden" }, () => null)
        .exhaustive(),
    )
    .with({ status: "loading" }, { status: "empty" }, () => null)
    .exhaustive();
}

function DeploymentPicker({
  value,
  choices,
  deployments,
  onValueChange,
}: {
  value: string;
  choices: TargetDeployment[];
  deployments: ListView<TargetDeployment>;
  onValueChange: (id: string) => void;
}) {
  const pager = pagerOf(deployments);
  return (
    <div className="flex items-start gap-2">
      <FormSelect
        aria-label="Deployment"
        size="sm"
        className="min-w-0 flex-1"
        placeholder="Choose a deployment"
        options={deploymentOptions(value, choices).map(deploymentSelectOption)}
        value={value}
        onValueChange={onValueChange}
        error={pager?.error ?? undefined}
      />
      {pager?.action}
    </div>
  );
}

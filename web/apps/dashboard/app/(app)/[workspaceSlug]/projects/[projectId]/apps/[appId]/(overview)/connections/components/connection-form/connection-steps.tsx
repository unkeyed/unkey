"use client";

import { IconCubeOutline18 } from "@unkey/icons";
import { Button } from "@unkey/ui";
import type { Dispatch, ReactNode, SetStateAction } from "react";
import { useFormContext, useFormState, useWatch } from "react-hook-form";
import { type ConnectionTargets, connectionHostVariable } from "../../connection-rules";
import type { AfterSave } from "../../hooks/use-connection-actions";
import type { TargetDeployments } from "../../hooks/use-connections-data";
import type { ConnectionFormValues } from "../../model/form";
import type { ConnectionEnvs } from "../../model/merge";
import {
  type FormStep,
  type StepsState,
  allStepsDone,
  completeStep,
  openStep,
  stepStatus,
} from "../../model/steps";
import { CodeSnippets } from "./code-snippets";
import { EnvironmentsCard } from "./environments-card";
import { StepCard } from "./step-card";
import { VariableField, VariableSummary } from "./variable-field";

type Mode = "add" | "edit";

const REDEPLOY_NOTE: Record<Mode, string | null> = {
  add: null,
  edit: "Changes take effect after a redeploy.",
};

const COMMIT_LABELS: Record<Mode, Record<AfterSave, string>> = {
  add: { prompt: "Add connection", redeploy: "Add and redeploy" },
  edit: { prompt: "Save changes", redeploy: "Save and redeploy" },
};

export function ConnectionSteps({
  mode,
  steps,
  onStepsChange,
  onCommit,
  committing,
  app,
  target,
  savedName,
  envs,
  targets,
  page,
  requiresExplicitPreviewTarget,
}: {
  mode: Mode;
  steps: StepsState;
  onStepsChange: Dispatch<SetStateAction<StepsState>>;
  onCommit: (after: AfterSave) => void;
  committing: AfterSave | null;
  app: { picker: ReactNode; summary: ReactNode; locked: boolean };
  target: { id: string; name: string } | undefined;
  savedName?: string;
  envs: ConnectionEnvs;
  targets: ConnectionTargets;
  page: TargetDeployments;
  requiresExplicitPreviewTarget: boolean;
}) {
  const { control, trigger } = useFormContext<ConnectionFormValues>();
  const name = useWatch({ control, name: "name" }).trim();
  const { isDirty } = useFormState({ control });
  const showCommit = mode === "add" || isDirty;
  const reopen = (step: FormStep) => () => onStepsChange((state) => openStep(state, step));
  const continueFromConfig = async () => {
    if (await trigger(["name", "envs"])) {
      onStepsChange(completeStep);
    }
  };

  return (
    <>
      <StepCard
        number={1}
        title="Select an app to connect to"
        label="App"
        status={stepStatus(steps, "app")}
        summary={app.summary}
        onOpen={app.locked ? undefined : reopen("app")}
      >
        {app.picker}
      </StepCard>
      <StepCard
        number={2}
        title="Configure"
        label="Variable"
        status={stepStatus(steps, "config")}
        summary={<VariableSummary />}
        onOpen={reopen("config")}
      >
        <VariableField savedName={savedName} />
        {target ? (
          <EnvironmentsCard
            envs={envs}
            targetAppId={target.id}
            targetAppName={target.name}
            targets={targets}
            page={page}
            requiresExplicitPreviewTarget={requiresExplicitPreviewTarget}
          />
        ) : null}
        <div className="flex items-center justify-between gap-2">
          {mode === "add" ? (
            <Button type="button" variant="outline" size="sm" onClick={reopen("app")}>
              Back
            </Button>
          ) : (
            <span />
          )}
          <Button
            type="button"
            variant="primary"
            size="sm"
            className="px-3"
            onClick={continueFromConfig}
          >
            Continue
          </Button>
        </div>
      </StepCard>
      <StepCard
        number={3}
        title="Connect your code"
        label="Code"
        status={allStepsDone(steps) && target ? "open" : "todo"}
      >
        {target ? (
          <CodeSnippets variable={connectionHostVariable(name)} targetAppName={target.name} />
        ) : null}
        {showCommit ? (
          <div className="flex justify-end gap-2">
            <Button
              type="button"
              variant="outline"
              size="sm"
              loading={committing === "prompt"}
              disabled={committing !== null}
              onClick={() => onCommit("prompt")}
            >
              {COMMIT_LABELS[mode].prompt}
            </Button>
            <Button
              type="button"
              variant="primary"
              size="sm"
              className="px-3"
              loading={committing === "redeploy"}
              disabled={committing !== null}
              onClick={() => onCommit("redeploy")}
            >
              {COMMIT_LABELS[mode].redeploy}
            </Button>
          </div>
        ) : null}
      </StepCard>
      {REDEPLOY_NOTE[mode] ? <p className="text-xs text-gray-10">{REDEPLOY_NOTE[mode]}</p> : null}
    </>
  );
}

export function AppName({ name }: { name: string }) {
  return (
    <span className="flex min-w-0 items-center gap-2">
      <IconCubeOutline18 className="size-3.5 shrink-0 text-gray-11" />
      <span className="truncate">{name}</span>
    </span>
  );
}

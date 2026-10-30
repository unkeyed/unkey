"use client";

import { IconCubeOutline18 } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { useFormContext, useFormState, useWatch } from "react-hook-form";
import type { ConnectionFormController } from "../../hooks/use-connection-form";
import type { ConnectionFormValues } from "../../model/form";
import { FORM_MODES } from "../../model/mode";
import { connectionHostVariable } from "../../model/name";
import { type FormStep, STEP_FIELDS, completeStep, openStep, stepStatus } from "../../model/steps";
import { AppPicker } from "./app-picker";
import { CodeSnippets } from "./code-snippets";
import { StepCard } from "./step-card";
import { TargetField } from "./target-field";
import { VariableField, VariableSummary } from "./variable-field";

export function ConnectionSteps({ controller }: { controller: ConnectionFormController }) {
  const { mode, before, steps, setSteps, target, pending, commit, pickApp } = controller;
  const { appLocked, commitLabel, commitNeedsChanges, redeployNote } = FORM_MODES[mode];
  const { control, trigger } = useFormContext<ConnectionFormValues>();
  const name = useWatch({ control, name: "name" }).trim();
  const { isDirty } = useFormState({ control });
  const showCommit = !commitNeedsChanges || isDirty;
  const reopen = (step: FormStep) => () => setSteps((state) => openStep(state, step));
  const continueFromConfig = async () => {
    if (await trigger([...STEP_FIELDS.config])) {
      setSteps(completeStep);
    }
  };

  return (
    <>
      <StepCard
        number={1}
        title="Select an app to connect to"
        label="App"
        status={stepStatus(steps, "app")}
        summary={target ? <AppName name={target.name} /> : null}
        onOpen={appLocked ? undefined : reopen("app")}
      >
        <AppPicker apps={controller.apps} connectedIds={controller.connectedIds} onPick={pickApp} />
      </StepCard>
      <StepCard
        number={2}
        title="Configure"
        label="Variable"
        status={stepStatus(steps, "config")}
        summary={<VariableSummary />}
        onOpen={reopen("config")}
      >
        <VariableField savedName={before?.name} />
        {controller.choices ? (
          <TargetField choices={controller.choices} deployments={controller.deployments} />
        ) : null}
        <div className="flex items-center justify-between gap-2">
          {appLocked ? (
            <span />
          ) : (
            <Button type="button" variant="outline" size="sm" onClick={reopen("app")}>
              Back
            </Button>
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
        status={stepStatus(steps, "code")}
      >
        {target ? (
          <CodeSnippets variable={connectionHostVariable(name)} targetAppName={target.name} />
        ) : null}
        {showCommit ? (
          <div className="flex justify-end">
            <Button
              type="button"
              variant="primary"
              size="sm"
              className="px-3"
              loading={pending?.action === mode}
              disabled={pending !== null}
              onClick={commit}
            >
              {commitLabel}
            </Button>
          </div>
        ) : null}
      </StepCard>
      {redeployNote ? <p className="text-xs text-gray-10">{redeployNote}</p> : null}
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

"use client";

import { IconCubeOutline18 } from "@unkey/icons";
import { Button } from "@unkey/ui";
import type { Dispatch, ReactNode, SetStateAction } from "react";
import { useFormContext, useFormState } from "react-hook-form";
import { type ConnectionTargets, connectionHostVariable } from "../../connection-rules";
import type { TargetDeployments } from "../../hooks/use-connections-data";
import type { ConnectionFormValues } from "../../model/form";
import type { ConnectionEnvs } from "../../model/merge";
import { type FormStep, type StepsState, openStep, stepStatus } from "../../model/steps";
import { CodeSnippets } from "./code-snippets";
import { EnvironmentsCard } from "./environments-card";
import { StepCard } from "./step-card";
import { VariableField, VariableSummary } from "./variable-field";

type Mode = "add" | "edit";

const SUBMIT_LABEL: Record<Mode, string> = { add: "Add connection", edit: "Save changes" };

export function ConnectionSteps({
  mode,
  steps,
  onStepsChange,
  created,
  onDone,
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
  created: boolean;
  onDone: () => void;
  app: { picker: ReactNode; summary: ReactNode; locked: boolean };
  target: { id: string; name: string } | undefined;
  savedName?: string;
  envs: ConnectionEnvs;
  targets: ConnectionTargets;
  page: TargetDeployments;
  requiresExplicitPreviewTarget: boolean;
}) {
  const { control, getValues } = useFormContext<ConnectionFormValues>();
  const { isSubmitting } = useFormState({ control });
  const reopen = (step: FormStep) => () => onStepsChange((state) => openStep(state, step));

  return (
    <>
      <StepCard
        number={1}
        title="Select an app"
        label="App"
        status={stepStatus(steps, "app")}
        summary={app.summary}
        onOpen={app.locked || created ? undefined : reopen("app")}
      >
        {app.picker}
      </StepCard>
      <StepCard
        number={2}
        title="Configure"
        label="Variable"
        status={stepStatus(steps, "config")}
        summary={<VariableSummary />}
        onOpen={created ? undefined : reopen("config")}
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
          <Button type="submit" variant="primary" size="sm" className="px-3" loading={isSubmitting}>
            {SUBMIT_LABEL[mode]}
          </Button>
        </div>
      </StepCard>
      {mode === "add" ? (
        <StepCard number={3} title="Complete" label="Complete" status={created ? "open" : "todo"}>
          {target ? (
            <CodeSnippets
              variable={connectionHostVariable(getValues("name").trim())}
              targetAppName={target.name}
            />
          ) : null}
          <div className="flex justify-end">
            <Button type="button" variant="primary" size="sm" className="px-3" onClick={onDone}>
              Done
            </Button>
          </div>
        </StepCard>
      ) : null}
      <p className="text-xs text-gray-10">Takes effect after a redeploy.</p>
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

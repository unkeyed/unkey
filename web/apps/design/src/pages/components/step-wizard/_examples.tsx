import { useState } from "react";
import { Button, StepWizard, useStepWizard } from "@unkey/ui";
import { Preview } from "../../../components/Preview";

function StepShell({
  title,
  children,
}: {
  title: string;
  children: React.ReactNode;
}) {
  const { activeStepIndex, totalSteps, next, back, canGoBack, isLastStep } =
    useStepWizard();

  return (
    <div className="flex w-full max-w-md flex-col gap-4 rounded-lg border border-grayA-3 bg-background p-5">
      <div className="flex items-center justify-between">
        <div className="text-xs uppercase tracking-[0.2em] text-gray-9">
          Step {activeStepIndex + 1} of {totalSteps}
        </div>
        <div className="text-sm font-medium text-gray-12">{title}</div>
      </div>
      <div className="min-h-[96px] text-sm text-gray-11">{children}</div>
      <div className="flex items-center justify-between">
        <Button variant="outline" onClick={back} disabled={!canGoBack}>
          Back
        </Button>
        <Button onClick={next}>{isLastStep ? "Finish" : "Continue"}</Button>
      </div>
    </div>
  );
}

export function BasicExample() {
  return (
    <Preview>
      <StepWizard.Root>
        <StepWizard.Step id="workspace" label="Workspace">
          <StepShell title="Workspace">
            Name the workspace that will hold your ACME keys. You can rename it
            later from settings.
          </StepShell>
        </StepWizard.Step>
        <StepWizard.Step id="api" label="API">
          <StepShell title="API">
            Pick a first API to issue keys against. The wizard scaffolds a
            default rate limit so you can verify straight away.
          </StepShell>
        </StepWizard.Step>
        <StepWizard.Step id="review" label="Review">
          <StepShell title="Review">
            Review the workspace and API settings, then finish to land on the
            dashboard.
          </StepShell>
        </StepWizard.Step>
      </StepWizard.Root>
    </Preview>
  );
}

function CompletedShell({
  title,
  children,
  done,
  onReset,
}: {
  title: string;
  children: React.ReactNode;
  done: boolean;
  onReset: () => void;
}) {
  const { activeStepIndex, totalSteps, next, back, canGoBack, isLastStep } =
    useStepWizard();

  if (done) {
    return (
      <div className="flex w-full max-w-md flex-col gap-4 rounded-lg border border-grayA-3 bg-background p-5">
        <div className="text-xs uppercase tracking-[0.2em] text-success-11">
          Complete
        </div>
        <div className="text-sm text-gray-11">
          The ACME workspace is provisioned. You can issue your first key from
          the dashboard.
        </div>
        <Button variant="outline" onClick={onReset}>
          Start over
        </Button>
      </div>
    );
  }

  return (
    <div className="flex w-full max-w-md flex-col gap-4 rounded-lg border border-grayA-3 bg-background p-5">
      <div className="flex items-center justify-between">
        <div className="text-xs uppercase tracking-[0.2em] text-gray-9">
          Step {activeStepIndex + 1} of {totalSteps}
        </div>
        <div className="text-sm font-medium text-gray-12">{title}</div>
      </div>
      <div className="min-h-[96px] text-sm text-gray-11">{children}</div>
      <div className="flex items-center justify-between">
        <Button variant="outline" onClick={back} disabled={!canGoBack}>
          Back
        </Button>
        <Button onClick={next}>{isLastStep ? "Finish" : "Continue"}</Button>
      </div>
    </div>
  );
}

export function CompletedStateExample() {
  const [done, setDone] = useState(false);
  const [runId, setRunId] = useState(0);

  return (
    <Preview>
      <StepWizard.Root
        key={runId}
        onComplete={() => setDone(true)}
      >
        <StepWizard.Step id="details" label="Details">
          <CompletedShell
            title="Details"
            done={done}
            onReset={() => {
              setDone(false);
              setRunId((n) => n + 1);
            }}
          >
            Give the ACME workspace a slug. We use it in generated URLs.
          </CompletedShell>
        </StepWizard.Step>
        <StepWizard.Step id="confirm" label="Confirm">
          <CompletedShell
            title="Confirm"
            done={done}
            onReset={() => {
              setDone(false);
              setRunId((n) => n + 1);
            }}
          >
            Press Finish to call <code>onComplete</code>, which flips this
            preview into its done state.
          </CompletedShell>
        </StepWizard.Step>
      </StepWizard.Root>
    </Preview>
  );
}

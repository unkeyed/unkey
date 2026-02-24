"use client";

import { StepWizard } from "@unkey/ui";
import { ProjectsListControls } from "./_components/controls";
import { ProjectsList } from "./_components/list";
import { ProjectsListNavigation } from "./navigation";

export function ProjectsClient() {
  if (!showWizard) {
    return (
      <div>
        <ProjectsListNavigation />
        <ProjectsListControls />
        <ProjectsList />
      </div>
    );
  }

  return (
    <StepWizard.Root
      onComplete={() => {
        /* redirect TBD */
      }}
    >
      <div className="relative flex-1 min-h-0">
        <StepWizard.Step id="step-1" label="Step 1" kind="optional">
          <div>step-1</div>
        </StepWizard.Step>
        <StepWizard.Step id="step-2" label="Step 2" kind="optional">
          <div>step-2</div>
        </StepWizard.Step>
        <StepWizard.Step id="step-3" label="Step 3" kind="optional">
          <div>step-3</div>
        </StepWizard.Step>
      </div>
    </StepWizard.Root>
  );
}

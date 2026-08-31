"use client";

import { OnboardingStepContainer } from "@/components/onboarding/step-container";
import { OnboardingStepHeader } from "@/components/onboarding/step-header";
import { match } from "@unkey/match";
import { StepWizard } from "@unkey/ui";
import { useState } from "react";
import { ChooseDestinationStep } from "./choose-destination-step";
import { ConfigureDestinationStep } from "./configure-destination-step";
import type { Kind } from "./form-schema";

export default function NewLogdrainPage() {
  const [kind, setKind] = useState<Kind | null>(null);

  return (
    <StepWizard.Root>
      <StepWizard.Step id="choose-destination" label="Choose destination">
        <OnboardingStepContainer>
          <OnboardingStepHeader
            title="Create log drain"
            showIconRow
            subtitle="Choose where to send audit logs."
          />
          <ChooseDestinationStep onSelect={setKind} />
        </OnboardingStepContainer>
      </StepWizard.Step>
      <StepWizard.Step id="configure-destination" label="Configure destination">
        {kind ? (
          <OnboardingStepContainer>
            <OnboardingStepHeader
              title={`Configure ${sinkName(kind)} destination`}
              subtitle="Enter the destination details."
              allowBack
            />
            <ConfigureDestinationStep key={kind} kind={kind} />
          </OnboardingStepContainer>
        ) : null}
      </StepWizard.Step>
    </StepWizard.Root>
  );
}

function sinkName(kind: Kind): string {
  return match(kind)
    .with("http", () => "HTTP")
    .with("axiom", () => "Axiom")
    .exhaustive();
}

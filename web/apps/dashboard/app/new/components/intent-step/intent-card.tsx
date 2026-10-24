import { Radio } from "@base-ui/react/radio";
import { RadioGroup } from "@base-ui/react/radio-group";
import { Button } from "@unkey/ui";
import { useId, useState } from "react";
import {
  OnboardingCard,
  OnboardingCardContent,
  OnboardingCardDescription,
  OnboardingCardFooter,
  OnboardingCardHeader,
  OnboardingCardTitle,
} from "../onboarding-card";
import { INTENTS, INTENT_ORDER, type Intent } from "./intents";

export function IntentCard({ onContinue }: { onContinue: (intent: Intent) => void }) {
  const [draft, setDraft] = useState<Intent | null>(null);
  const titleId = useId();

  return (
    <OnboardingCard>
      <OnboardingCardHeader>
        <OnboardingCardTitle id={titleId}>What brings you to Unkey?</OnboardingCardTitle>
        <OnboardingCardDescription>
          We'll set up your workspace for this first.
        </OnboardingCardDescription>
      </OnboardingCardHeader>
      <OnboardingCardContent>
        <RadioGroup
          aria-labelledby={titleId}
          value={draft}
          onValueChange={setDraft}
          className="grid gap-3 sm:grid-cols-2"
        >
          {INTENT_ORDER.map((intent) => {
            const { title, description, icon: Icon } = INTENTS[intent];
            return (
              <Radio.Root
                key={intent}
                value={intent}
                className="group relative flex flex-col items-center gap-3 rounded-lg border border-grayA-5 px-4 pt-8 pb-6 text-center transition-colors duration-150 ease-out hover:border-grayA-7 focus:outline-hidden focus-visible:ring-2 focus-visible:ring-gray-7 data-checked:border-gray-12 data-checked:bg-grayA-2 data-checked:ring-1 data-checked:ring-gray-12 data-checked:focus-visible:ring-2 data-checked:focus-visible:ring-gray-7"
              >
                <span className="absolute top-3 right-3 flex size-4 items-center justify-center rounded-full border border-grayA-7 group-data-checked:border-gray-12 group-data-checked:bg-gray-12">
                  <Radio.Indicator className="size-1.5 rounded-full bg-gray-1" />
                </span>
                <span
                  aria-hidden="true"
                  className="flex size-12 items-center justify-center rounded-full border bg-gray-2 text-gray-11 group-data-checked:text-gray-12"
                >
                  <Icon className="size-5" />
                </span>
                <span className="flex flex-col gap-1">
                  <span className="text-sm font-medium text-gray-12">{title}</span>
                  <span className="text-xs leading-5 text-gray-11">{description}</span>
                </span>
              </Radio.Root>
            );
          })}
        </RadioGroup>
      </OnboardingCardContent>
      <OnboardingCardFooter>
        <Button
          size="xlg"
          className="w-full rounded-lg"
          onClick={() => {
            if (draft !== null) {
              onContinue(draft);
            }
          }}
          disabled={draft === null}
        >
          Continue
        </Button>
      </OnboardingCardFooter>
    </OnboardingCard>
  );
}

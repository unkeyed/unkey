"use client";
import { HelpButton } from "@/components/navigation/top-nav/help-button";
import { signOut } from "@/lib/auth/utils";
import { useQueryClient } from "@tanstack/react-query";
import { Button, FullScreenContent, FullScreenLayout, Logo } from "@unkey/ui";
import { useState } from "react";
import { useWorkspaceStep } from "../hooks/use-workspace-step";
import { IntentCard } from "./intent-step/intent-card";
import { INTENTS, type Intent } from "./intent-step/intents";
import {
  OnboardingCard,
  OnboardingCardContent,
  OnboardingCardDescription,
  OnboardingCardFooter,
  OnboardingCardHeader,
  OnboardingCardTitle,
} from "./onboarding-card";

export function OnboardingContent() {
  const [intent, setIntent] = useState<Intent | null>(null);
  const queryClient = useQueryClient();

  return (
    <FullScreenLayout className="px-4 pt-6">
      <div className="absolute top-4 right-4">
        <Button
          variant="ghost"
          size="sm"
          onClick={async () => {
            queryClient.clear();
            await signOut();
          }}
          className="text-gray-11"
        >
          Sign out
        </Button>
      </div>

      <Logo />

      <FullScreenContent className="max-w-sm py-10 sm:max-w-md lg:max-w-lg">
        {intent === null ? (
          <IntentCard onContinue={setIntent} />
        ) : (
          <WorkspaceCard intent={intent} />
        )}
      </FullScreenContent>
      <div className="absolute bottom-4 right-4">
        <HelpButton />
      </div>
    </FullScreenLayout>
  );
}

function WorkspaceCard({ intent }: { intent: Intent }) {
  const { body, submit, isDisabled, isLoading } = useWorkspaceStep(INTENTS[intent].landing);

  return (
    <OnboardingCard>
      <OnboardingCardHeader>
        <OnboardingCardTitle>Create Company Workspace</OnboardingCardTitle>
        <OnboardingCardDescription>
          Name your workspace and choose its URL.
        </OnboardingCardDescription>
      </OnboardingCardHeader>
      <OnboardingCardContent>{body}</OnboardingCardContent>
      <OnboardingCardFooter>
        <Button
          size="xlg"
          className="w-full rounded-lg"
          onClick={submit}
          disabled={isDisabled}
          loading={isLoading}
        >
          Create workspace
        </Button>
      </OnboardingCardFooter>
    </OnboardingCard>
  );
}

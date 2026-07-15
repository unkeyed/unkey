"use client";
import { HelpButton } from "@/components/navigation/top-nav/help-button";
import { signOut } from "@/lib/auth/utils";
import { useQueryClient } from "@tanstack/react-query";
import { Button, FullScreenContent, FullScreenLayout, Logo } from "@unkey/ui";
import Link from "next/link";
import { useWorkspaceStep } from "../hooks/use-workspace-step";
import {
  OnboardingCard,
  OnboardingCardContent,
  OnboardingCardDescription,
  OnboardingCardFooter,
  OnboardingCardHeader,
  OnboardingCardTitle,
} from "./onboarding-card";

export function OnboardingContent() {
  const { body, submit, isDisabled, isLoading } = useWorkspaceStep();
  const queryClient = useQueryClient();

  return (
    <FullScreenLayout className="overflow-x-hidden bg-gray-2 dark:bg-background">
      <nav className="flex items-center justify-between h-16 w-full shrink-0 px-6">
        <Link href="/">
          <Logo />
        </Link>
        <Button
          variant="outline"
          size="md"
          className="font-medium"
          onClick={async () => {
            queryClient.clear();
            await signOut();
          }}
        >
          Sign out
        </Button>
      </nav>

      <FullScreenContent className="px-4 py-8">
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
      </FullScreenContent>
      <div className="absolute bottom-4 right-4">
        <HelpButton />
      </div>
    </FullScreenLayout>
  );
}

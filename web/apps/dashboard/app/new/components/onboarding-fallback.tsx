"use client";
import { FullScreenContent, FullScreenLayout, Logo, Skeleton } from "@unkey/ui";
import {
  OnboardingCard,
  OnboardingCardContent,
  OnboardingCardFooter,
  OnboardingCardHeader,
} from "./onboarding-card";

export function OnboardingFallback() {
  return (
    <FullScreenLayout className="px-4 pt-6">
      <Logo />

      <FullScreenContent className="max-w-sm py-10 sm:max-w-md lg:max-w-lg">
        <OnboardingCard aria-busy="true">
          <OnboardingCardHeader>
            <Skeleton className="h-7 w-1/2" />
            <Skeleton className="h-5 w-3/4 mt-2" />
          </OnboardingCardHeader>
          <OnboardingCardContent>
            <div className="flex flex-col gap-4">
              <div className="flex flex-col gap-1.5">
                <Skeleton className="h-4 w-1/4" />
                <Skeleton className="h-9 w-full rounded-lg" />
              </div>
              <div className="flex flex-col gap-1.5">
                <Skeleton className="h-4 w-1/3" />
                <Skeleton className="h-9 w-full rounded-lg" />
              </div>
            </div>
          </OnboardingCardContent>
          <OnboardingCardFooter>
            <Skeleton className="h-10 w-full rounded-lg" />
          </OnboardingCardFooter>
        </OnboardingCard>
      </FullScreenContent>
    </FullScreenLayout>
  );
}

"use client";
import { FullScreenContent, FullScreenLayout, Logo, Skeleton } from "@unkey/ui";
import Link from "next/link";
import {
  OnboardingCard,
  OnboardingCardContent,
  OnboardingCardFooter,
  OnboardingCardHeader,
} from "./onboarding-card";

export function OnboardingFallback() {
  return (
    <FullScreenLayout className="overflow-x-hidden bg-gray-2 dark:bg-background">
      <nav className="flex items-center justify-between h-16 w-full shrink-0 px-6">
        <Link href="/">
          <Logo />
        </Link>
        {/* Placeholder matching the outline "Sign out" button (size md = h-8). */}
        <Skeleton className="h-8 w-20 rounded-md" />
      </nav>

      <FullScreenContent className="px-4 py-8">
        <OnboardingCard aria-busy="true">
          <OnboardingCardHeader>
            <Skeleton className="h-8 w-1/2" />
            <Skeleton className="h-5 w-3/4" />
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

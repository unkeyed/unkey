"use server";
import { getAuth } from "@/lib/auth/get-auth";
import { Suspense } from "react";
import { OnboardingContent } from "./components/onboarding-content";
import { OnboardingFallback } from "./components/onboarding-fallback";

export default async function OnboardingPage() {
  // Ensure we have an authenticated user before proceeding
  // We don't actually need any user data though
  await getAuth();

  return (
    <Suspense fallback={<OnboardingFallback />}>
      <OnboardingContent />
    </Suspense>
  );
}

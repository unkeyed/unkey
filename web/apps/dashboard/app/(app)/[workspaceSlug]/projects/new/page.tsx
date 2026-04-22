"use server";

import { LoadingState } from "@/components/loading-state";
import { Suspense } from "react";
import { Onboarding } from "./index";

export default async function OnboardingPage() {
  return (
    <Suspense fallback={<LoadingState message="Loading projects onboarding..." />}>
      <Onboarding />
    </Suspense>
  );
}

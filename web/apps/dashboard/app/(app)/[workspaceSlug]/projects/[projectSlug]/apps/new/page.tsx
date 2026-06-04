"use server";

import { LoadingState } from "@/components/loading-state";
import { Suspense } from "react";
import { AppSetupWizard } from "./index";

export default async function AppSetupPage() {
  return (
    <Suspense fallback={<LoadingState message="Loading app setup..." />}>
      <AppSetupWizard />
    </Suspense>
  );
}

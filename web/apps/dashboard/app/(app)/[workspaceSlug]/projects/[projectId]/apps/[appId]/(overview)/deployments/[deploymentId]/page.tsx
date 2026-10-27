"use client";

import { useFlag } from "@/lib/flags/provider";
import { DeploymentOverview } from "./deployment-overview";
import { DeploymentOverview as LegacyDeploymentOverview } from "./legacy/deployment-overview";

export default function DeploymentPage() {
  return useFlag("appV2") ? <DeploymentOverview /> : <LegacyDeploymentOverview />;
}

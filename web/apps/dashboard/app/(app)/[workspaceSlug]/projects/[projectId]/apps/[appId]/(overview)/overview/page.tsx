"use client";

import { useFlag } from "@/lib/flags/provider";
import { Overview } from "./app-overview";
import { Overview as LegacyOverview } from "./legacy/app-overview";

export default function OverviewPage() {
  return useFlag("appV2") ? <Overview /> : <LegacyOverview />;
}

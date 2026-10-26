"use client";

import { NEXT_DEPLOY } from "@/lib/collections/deploy/pending-redeploy";
import { SettingsGroups } from "@unkey/ui";
import { useParams } from "next/navigation";
import type { ReactNode } from "react";
import { AdvancedSettingsSkeleton } from "../../../../_components/settings/advanced-settings/advanced-settings";
import { EnvironmentSettingsProvider } from "../../../../_components/settings/environment-provider";
import { SettingsSkeleton } from "../../../../_components/settings/settings-skeleton";

const SKELETONS: Record<string, ReactNode> = {
  advanced: <AdvancedSettingsSkeleton />,
};

export default function EnvironmentSettingsLayout({ children }: { children: ReactNode }) {
  const { page } = useParams<{ page: string }>();
  return (
    <EnvironmentSettingsProvider fallback={SKELETONS[page] ?? <SettingsSkeleton />}>
      <SettingsGroups pendingNote={NEXT_DEPLOY}>{children}</SettingsGroups>
    </EnvironmentSettingsProvider>
  );
}

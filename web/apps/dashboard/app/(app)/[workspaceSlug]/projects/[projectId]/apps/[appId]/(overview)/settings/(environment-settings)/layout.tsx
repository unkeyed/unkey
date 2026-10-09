"use client";

import { NEXT_DEPLOY } from "@/lib/collections/deploy/pending-redeploy";
import { SettingsGroups } from "@unkey/ui";
import type { ReactNode } from "react";
import { EnvironmentSettingsProvider } from "../../../../_components/settings/environment-provider";

export default function EnvironmentSettingsLayout({ children }: { children: ReactNode }) {
  return (
    <EnvironmentSettingsProvider>
      <SettingsGroups pendingNote={NEXT_DEPLOY}>{children}</SettingsGroups>
    </EnvironmentSettingsProvider>
  );
}

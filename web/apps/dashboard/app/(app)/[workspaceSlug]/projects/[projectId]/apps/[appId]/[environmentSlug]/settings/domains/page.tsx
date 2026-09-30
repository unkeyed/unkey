"use client";

import { SettingsGroups } from "@unkey/ui";
import { CustomDomains } from "../components/domains/custom-domains";
import { PlatformDomains } from "../components/domains/platform-domains";

export default function DomainsSettingsPage() {
  return (
    <SettingsGroups>
      <PlatformDomains />
      <CustomDomains />
    </SettingsGroups>
  );
}

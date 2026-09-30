"use client";

import type { Domain } from "@/lib/collections/deploy/domains";
import { IconLink4Outline18 } from "@unkey/icons";
import { CopyButton, SettingCardGroup, SettingsRow } from "@unkey/ui";
import { useProjectData } from "../../../../data-provider";
import { useAppEnvironment } from "../../../environment-context";
import { SettingsSection } from "../shared/settings-section";

const PLATFORM_DOMAIN_ROWS: ReadonlyArray<{
  sticky: Domain["sticky"];
  title: string;
  description: string;
}> = [
  {
    sticky: "live",
    title: "Live URL",
    description: "Always points at the live deployment. Unkey manages it.",
  },
  {
    sticky: "environment",
    title: "Environment URL",
    description: "Points at the latest deployment in this environment. Unkey manages it.",
  },
];

export function PlatformDomains() {
  const { domains } = useProjectData();
  const { environment } = useAppEnvironment();

  const rows = PLATFORM_DOMAIN_ROWS.flatMap((row) =>
    domains
      .filter((domain) => domain.environmentId === environment.id && domain.sticky === row.sticky)
      .map((domain) => ({ ...row, hostname: domain.fullyQualifiedDomainName })),
  );

  return (
    <SettingsSection title="Platform domain">
      <SettingCardGroup>
        {rows.length === 0 ? (
          <p className="px-5 py-6 text-center text-xs text-gray-10">
            Unkey creates a platform domain on the first deployment.
          </p>
        ) : (
          rows.map((row) => (
            <SettingsRow key={row.hostname} title={row.title} description={row.description}>
              <div className="flex max-w-(--setting-w) items-center gap-2">
                <div className="flex h-9 min-w-0 flex-1 items-center gap-2 rounded-lg border bg-grayA-2 px-3 font-mono text-xs text-gray-11">
                  <IconLink4Outline18 className="size-3.5 shrink-0 text-gray-9" />
                  <span className="truncate">{row.hostname}</span>
                </div>
                <CopyButton value={row.hostname} toastMessage={row.hostname} className="size-9" />
              </div>
            </SettingsRow>
          ))
        )}
      </SettingCardGroup>
    </SettingsSection>
  );
}

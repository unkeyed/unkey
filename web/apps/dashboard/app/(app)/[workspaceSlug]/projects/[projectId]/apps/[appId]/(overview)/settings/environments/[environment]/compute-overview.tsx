"use client";

import { NEXT_DEPLOY } from "@/lib/collections/deploy/pending-redeploy";
import { trpc } from "@/lib/trpc/client";
import { IconGearOutline12 } from "@unkey/icons";
import {
  Button,
  SettingsGroup,
  SettingsGroupContent,
  SettingsGroupTitle,
  SettingsPanel,
} from "@unkey/ui";
import { useState } from "react";
import { ComputeForm } from "../../../../../_components/settings/compute/compute-settings";
import { readCompute } from "../../../../../_components/settings/compute/draft";
import { RegionViz } from "../../../../../_components/settings/compute/region-viz";
import { availableFrom, unschedulableIn } from "../../../../../_components/settings/compute/status";
import { useEnvironmentSettings } from "../../../../../_components/settings/environment-provider";

export function ComputeOverview() {
  const { settings } = useEnvironmentSettings();
  const regionsQuery = trpc.deploy.environmentSettings.getAvailableRegions.useQuery();
  const [configuring, setConfiguring] = useState(false);
  const close = () => setConfiguring(false);

  return (
    <SettingsGroup>
      <SettingsGroupTitle>Compute</SettingsGroupTitle>
      <SettingsGroupContent>
        <RegionViz
          settings={settings}
          action={
            <Button
              variant="outline"
              size="sm"
              className="bg-raised"
              onClick={() => setConfiguring(true)}
            >
              <IconGearOutline12 />
              Configure
            </Button>
          }
          unavailable={
            new Set(unschedulableIn(availableFrom(regionsQuery), readCompute(settings).regions))
          }
        />
      </SettingsGroupContent>
      <SettingsPanel
        isOpen={configuring}
        onClose={close}
        onSaved={close}
        title="Configure compute"
        description={`${NEXT_DEPLOY}.`}
      >
        <ComputeForm />
      </SettingsPanel>
    </SettingsGroup>
  );
}

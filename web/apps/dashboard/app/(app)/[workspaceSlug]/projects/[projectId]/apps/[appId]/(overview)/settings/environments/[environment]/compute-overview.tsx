"use client";

import { NEXT_DEPLOY } from "@/lib/collections/deploy/pending-redeploy";
import { trpc } from "@/lib/trpc/client";
import { IconGearOutline12 } from "@unkey/icons";
import {
  Button,
  SettingsGroup,
  SettingsGroupContent,
  SettingsGroupTitle,
  SlidePanel,
  SlidePanelClose,
  SlidePanelCloseButton,
  SlidePanelContent,
  SlidePanelDescription,
  SlidePanelFooter,
  SlidePanelHeader,
  SlidePanelTitle,
  useReportUnsavedChanges,
} from "@unkey/ui";
import { useState } from "react";
import { ComputeFields } from "../../../../../_components/settings/compute/compute-fields";
import { readCompute } from "../../../../../_components/settings/compute/draft";
import { availableFrom, unschedulableIn } from "../../../../../_components/settings/compute/status";
import { useCompute } from "../../../../../_components/settings/compute/use-compute";
import { useEnvironmentSettings } from "../../../../../_components/settings/environment-provider";
import { RegionViz } from "../../components/compute/region-viz";

export function ComputeOverview() {
  const { settings } = useEnvironmentSettings();
  const regionsQuery = trpc.deploy.environmentSettings.getAvailableRegions.useQuery();
  const values = readCompute(settings);
  const [configuring, setConfiguring] = useState(false);
  const close = () => setConfiguring(false);

  return (
    <SettingsGroup>
      <div className="flex items-center justify-between">
        <SettingsGroupTitle>Compute</SettingsGroupTitle>
        <Button variant="outline" size="sm" onClick={() => setConfiguring(true)}>
          <IconGearOutline12 />
          Configure
        </Button>
      </div>
      <SettingsGroupContent>
        <RegionViz
          values={values}
          unavailable={new Set(unschedulableIn(availableFrom(regionsQuery), values.regions))}
        />
      </SettingsGroupContent>
      <SlidePanel isOpen={configuring} onClose={close}>
        <SlidePanelHeader>
          <div className="flex flex-col gap-0.5">
            <SlidePanelTitle>Configure compute</SlidePanelTitle>
            <SlidePanelDescription>{NEXT_DEPLOY}.</SlidePanelDescription>
          </div>
          <SlidePanelCloseButton className="mt-0.5" />
        </SlidePanelHeader>
        <SlidePanelContent className="flex flex-col">
          <ConfigureComputeForm onSaved={close} />
        </SlidePanelContent>
      </SlidePanel>
    </SettingsGroup>
  );
}

function ConfigureComputeForm({ onSaved }: { onSaved: () => void }) {
  const compute = useCompute();
  const { saveState, dirty } = compute.formProps;
  useReportUnsavedChanges(dirty);

  return (
    <form
      onSubmit={async (event) => {
        if (await compute.submit(event)) {
          onSaved();
        }
      }}
      className="flex min-h-0 flex-1 flex-col"
    >
      <div className="flex-1 overflow-y-auto px-6 pt-4 pb-8">
        <ComputeFields compute={compute} />
      </div>
      <SlidePanelFooter className="flex items-center justify-end gap-3">
        <SlidePanelClose
          disabled={saveState.status === "saving"}
          render={<Button type="button" variant="outline" size="md" />}
        >
          Cancel
        </SlidePanelClose>
        <Button
          type="submit"
          variant="primary"
          size="md"
          disabled={saveState.status === "disabled"}
          loading={saveState.status === "saving"}
        >
          Save
        </Button>
      </SlidePanelFooter>
    </form>
  );
}

"use client";

import { RegionFlag } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/[appId]/components/region-flag";
import {
  Multibox,
  MultiboxChip,
  MultiboxChipRemove,
  MultiboxChips,
  MultiboxContent,
  MultiboxEmpty,
  MultiboxInput,
  MultiboxItem,
  MultiboxList,
  MultiboxTrigger,
  useMultiboxAnchor,
} from "@/components/ui/multibox";
import { trpc } from "@/lib/trpc/client";
import { mapRegionToFlag } from "@/lib/trpc/routers/deploy/network/utils";
import { zodResolver } from "@hookform/resolvers/zod";
import { IconLocation2Outline18 } from "@unkey/icons";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@unkey/ui";
import { FormLabel } from "@unkey/ui/src/components/form/form-helpers";
import { useEffect, useId, useMemo } from "react";
import { useForm, useWatch } from "react-hook-form";
import { z } from "zod";
import { useEnvironmentSettings } from "../../environment-provider";
import { useUpdateEnvironment } from "../../hooks/use-update-environment";
import { SettingDescription, SettingField } from "../shared/form-blocks";
import { FormSettingCard, resolveSaveState } from "../shared/form-setting-card";

const RegionMultibox = ({
  inputId,
  regions,
  onChange,
  availableRegions,
}: {
  inputId?: string;
  regions: string[];
  onChange: (regions: string[]) => void;
  availableRegions: Array<{ name: string; canSchedule: boolean }>;
}) => {
  const schedulableRegions = availableRegions.filter((r) => r.canSchedule).map((r) => r.name);
  const unschedulableRegions = new Set(
    availableRegions.filter((r) => !r.canSchedule).map((r) => r.name),
  );
  const canRemove = regions.length > 1;
  const anchor = useMultiboxAnchor();

  return (
    <TooltipProvider>
      <Multibox
        items={schedulableRegions}
        value={regions}
        onValueChange={(next) => {
          if (next.length > 0) {
            onChange(next);
          }
        }}
      >
        <MultiboxChips ref={anchor}>
          {regions.map((region) => {
            const isUnschedulable = unschedulableRegions.has(region);
            const chip = (
              <MultiboxChip
                key={region}
                className={
                  isUnschedulable ? "bg-warning-3 border-warning-6 text-warning-11" : undefined
                }
              >
                <RegionFlag
                  flagCode={mapRegionToFlag(region)}
                  size="xs"
                  shape="circle"
                  className="[&_img]:size-3"
                />
                {region}
                {canRemove && <MultiboxChipRemove />}
              </MultiboxChip>
            );

            if (isUnschedulable) {
              return (
                <Tooltip key={region}>
                  <TooltipTrigger render={chip} />
                  <TooltipContent>
                    This region is currently unavailable for scheduling
                  </TooltipContent>
                </Tooltip>
              );
            }
            return chip;
          })}
          <MultiboxInput
            id={inputId}
            placeholder={regions.length === 0 ? "Select a region" : undefined}
          />
          <MultiboxTrigger />
        </MultiboxChips>
        <MultiboxContent anchor={anchor}>
          <MultiboxEmpty>No regions available.</MultiboxEmpty>
          <MultiboxList>
            {(region: string) => (
              <MultiboxItem key={region} value={region}>
                <RegionFlag
                  flagCode={mapRegionToFlag(region)}
                  size="xs"
                  className="[&_img]:size-3"
                />
                <span className="text-gray-11 text-xs font-mono">{region}</span>
              </MultiboxItem>
            )}
          </MultiboxList>
        </MultiboxContent>
      </Multibox>
    </TooltipProvider>
  );
};

const RegionDisplayValue = ({ regions }: { regions: string[] }) => {
  if (regions.length === 0) {
    return null;
  }
  if (regions.length <= 2) {
    return (
      <span className="flex items-center gap-1.5">
        {regions.map((r, i) => (
          <span key={r} className="flex items-center gap-1.5">
            {i > 0 && <span className="text-grayA-4">|</span>}
            <span className="flex items-center gap-1">
              <RegionFlag
                flagCode={mapRegionToFlag(r)}
                size="xs"
                shape="circle"
                className="[&_img]:size-3"
              />
              <span className="text-gray-11">{r}</span>
            </span>
          </span>
        ))}
      </span>
    );
  }
  return (
    <span className="flex items-center gap-1">
      {regions.map((r) => (
        <RegionFlag key={r} flagCode={mapRegionToFlag(r)} size="xs" shape="circle" />
      ))}
    </span>
  );
};

const regionsSchema = z.object({
  regions: z.array(z.string()).min(1, "Select at least one region"),
});

type RegionsFormValues = z.infer<typeof regionsSchema>;

export const Regions = () => {
  const { settings, variant } = useEnvironmentSettings();
  const updateEnvironment = useUpdateEnvironment();
  const { environmentId, regions: settingsRegions } = settings;
  const defaultRegions = useMemo(() => settingsRegions.map((r) => r.name), [settingsRegions]);

  const { data: availableRegions } = trpc.deploy.environmentSettings.getAvailableRegions.useQuery(
    undefined,
    { enabled: Boolean(environmentId) },
  );

  const {
    handleSubmit,
    setValue,
    formState: { isValid, isSubmitting },
    control,
    reset,
  } = useForm<RegionsFormValues>({
    resolver: zodResolver(regionsSchema),
    mode: "onChange",
    defaultValues: { regions: defaultRegions },
  });

  useEffect(() => {
    reset({ regions: defaultRegions });
  }, [defaultRegions, reset]);

  const currentRegions = useWatch({ control, name: "regions" });
  const inputId = useId();

  const onSubmit = async (values: RegionsFormValues) => {
    updateEnvironment((draft) => {
      const defaultReplicasMin = draft.regions.at(0)?.replicasMin ?? 1;
      const defaultReplicasMax = draft.regions.at(0)?.replicasMax ?? 1;
      draft.regions = values.regions.map(
        (name) =>
          draft.regions.find((r) => r.name === name) ?? {
            name,
            replicasMin: defaultReplicasMin,
            replicasMax: defaultReplicasMax,
          },
      );
    });
  };

  const hasChanges =
    currentRegions.length !== defaultRegions.length ||
    currentRegions.some((r) => !defaultRegions.includes(r));

  const saveState = resolveSaveState([
    [isSubmitting, { status: "saving" }],
    [!isValid, { status: "disabled" }],
    [!hasChanges, { status: "disabled", reason: "No changes to save" }],
  ]);

  return (
    <FormSettingCard
      icon={<IconLocation2Outline18 className="text-gray-12" />}
      title="Regions"
      description="Geographic regions where your app will run"
      displayValue={<RegionDisplayValue regions={defaultRegions} />}
      onSubmit={handleSubmit(onSubmit)}
      saveState={saveState}
      autoSave={variant === "onboarding"}
    >
      <SettingField>
        <fieldset className="flex flex-col gap-1.5 border-0 m-0 p-0">
          <FormLabel label="Region" requirement="optional" htmlFor={inputId} />
          <RegionMultibox
            inputId={inputId}
            regions={currentRegions}
            onChange={(next) => setValue("regions", next, { shouldValidate: true })}
            availableRegions={availableRegions ?? []}
          />
        </fieldset>
      </SettingField>

      <SettingDescription>
        Traffic is routed to the nearest selected region. Changes apply on next deploy.
      </SettingDescription>
    </FormSettingCard>
  );
};

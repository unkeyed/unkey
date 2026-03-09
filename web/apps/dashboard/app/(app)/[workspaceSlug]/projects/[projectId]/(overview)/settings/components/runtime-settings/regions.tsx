"use client";

import type { ComboboxOption } from "@/components/ui/combobox";
import { FormCombobox } from "@/components/ui/form-combobox";
import { collection } from "@/lib/collections";
import { trpc } from "@/lib/trpc/client";
import { mapRegionToFlag } from "@/lib/trpc/routers/deploy/network/utils";
import { zodResolver } from "@hookform/resolvers/zod";
import { Location2, XMark } from "@unkey/icons";
import { useEffect } from "react";
import { useForm, useWatch } from "react-hook-form";
import { z } from "zod";
import { RegionFlag } from "../../../../components/region-flag";
import { useEnvironmentSettings } from "../../environment-provider";
import { FormSettingCard, resolveSaveState } from "../shared/form-setting-card";

const regionsSchema = z.object({
  regions: z.array(z.string()).min(1, "Select at least one region"),
});

type RegionsFormValues = z.infer<typeof regionsSchema>;

type AvailableRegion = { id: string; name: string };

export const Regions = () => {
  const { settings, autoSave } = useEnvironmentSettings();
  const { environmentId, regions } = settings;
  const defaultRegionNames = regions.map((r) => r.name);

  const { data: availableRegions } = trpc.deploy.environmentSettings.getAvailableRegions.useQuery(
    undefined,
    { enabled: Boolean(environmentId) },
  );

  return (
    <RegionsForm
      environmentId={environmentId}
      defaultRegionNames={defaultRegionNames}
      availableRegions={availableRegions ?? []}
      autoSave={autoSave}
    />
  );
};

type RegionsFormProps = {
  environmentId: string;
  defaultRegionNames: string[];
  availableRegions: AvailableRegion[];
  autoSave?: boolean;
};

const RegionsForm: React.FC<RegionsFormProps> = ({
  environmentId,
  defaultRegionNames,
  availableRegions,
  autoSave,
}) => {
  const {
    handleSubmit,
    setValue,
    formState: { isValid, isSubmitting },
    control,
    reset,
  } = useForm<RegionsFormValues>({
    resolver: zodResolver(regionsSchema),
    mode: "onChange",
    defaultValues: { regions: defaultRegionNames },
  });

  useEffect(() => {
    reset({ regions: defaultRegionNames });
  }, [defaultRegionNames, reset]);

  const currentRegions = useWatch({ control, name: "regions" });

  const unselectedRegions = availableRegions.filter((r) => !currentRegions.includes(r.name));

  const onSubmit = async (values: RegionsFormValues) => {
    collection.environmentSettings.update(environmentId, (draft) => {
      const defaultReplicas = draft.regions.at(0)?.replicas ?? 1;
      draft.regions = values.regions.map((name) => {
        const existing = draft.regions.find((r) => r.name === name);
        if (existing) {
          return existing;
        }
        const available = availableRegions.find((r) => r.name === name);
        return { id: available?.id ?? name, name, replicas: defaultReplicas };
      });
    });
  };

  const addRegion = (regionName: string) => {
    if (regionName && !currentRegions.includes(regionName)) {
      setValue("regions", [...currentRegions, regionName], { shouldValidate: true });
    }
  };

  const removeRegion = (regionName: string) => {
    setValue(
      "regions",
      currentRegions.filter((r) => r !== regionName),
      { shouldValidate: true },
    );
  };

  const hasChanges =
    currentRegions.length !== defaultRegionNames.length ||
    currentRegions.some((r) => !defaultRegionNames.includes(r));

  const saveState = resolveSaveState([
    [isSubmitting, { status: "saving" }],
    [!isValid, { status: "disabled" }],
    [!hasChanges, { status: "disabled", reason: "No changes to save" }],
  ]);

  const displayValue =
    defaultRegionNames.length === 0 ? null : defaultRegionNames.length <= 2 ? (
      <span className="flex items-center gap-1.5">
        {defaultRegionNames.map((r, i) => (
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
    ) : (
      <span className="flex items-center gap-1">
        {defaultRegionNames.map((r) => (
          <RegionFlag key={r} flagCode={mapRegionToFlag(r)} size="xs" shape="circle" />
        ))}
      </span>
    );

  const comboboxOptions: ComboboxOption[] = unselectedRegions.map((region) => ({
    value: region.name,
    searchValue: region.name,
    label: (
      <div className="flex items-center gap-2">
        <RegionFlag flagCode={mapRegionToFlag(region.name)} size="xs" className="[&_img]:size-3" />
        <span className="text-gray-11 text-xs font-mono">{region.name}</span>
      </div>
    ),
  }));

  return (
    <FormSettingCard
      icon={<Location2 className="text-gray-12" iconSize="xl-medium" />}
      title="Regions"
      description="Geographic regions where your project will run"
      displayValue={displayValue}
      onSubmit={handleSubmit(onSubmit)}
      saveState={saveState}
      autoSave={autoSave}
    >
      <FormCombobox
        label="Regions"
        description="Traffic is routed to the nearest selected region. Changes apply on next deploy."
        optional
        className="w-[480px]"
        options={comboboxOptions}
        value=""
        onSelect={addRegion}
        placeholder={
          currentRegions.length === 0 ? (
            <span className="text-grayA-8 w-full text-left">Select a region</span>
          ) : (
            <div className="w-full flex flex-wrap gap-1.5 py-0.5">
              {currentRegions.map((r) => (
                <span
                  key={r}
                  className="flex items-center gap-1 px-1.5 py-0.5 rounded-md bg-grayA-3 border border-grayA-4 text-xs text-accent-12"
                >
                  <RegionFlag
                    flagCode={mapRegionToFlag(r)}
                    size="xs"
                    shape="circle"
                    className="[&_img]:size-3"
                  />
                  {r}
                  {currentRegions.length > 1 && (
                    //biome-ignore lint/a11y/useKeyWithClickEvents: we can't use button here otherwise we'll nest two buttons
                    <span
                      onClick={(e) => {
                        e.stopPropagation();
                        removeRegion(r);
                      }}
                      className="p-0.5 hover:bg-grayA-4 rounded text-grayA-9 hover:text-accent-12 transition-colors"
                    >
                      <XMark iconSize="sm-regular" />
                    </span>
                  )}
                </span>
              ))}
            </div>
          )
        }
        searchPlaceholder="Search regions..."
        emptyMessage={<div className="mt-2">No regions available.</div>}
      />
    </FormSettingCard>
  );
};

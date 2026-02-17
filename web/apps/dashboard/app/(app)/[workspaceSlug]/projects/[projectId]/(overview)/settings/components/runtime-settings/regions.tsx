"use client";

import type { ComboboxOption } from "@/components/ui/combobox";
import { FormCombobox } from "@/components/ui/form-combobox";
import { trpc } from "@/lib/trpc/client";
import { mapRegionToFlag } from "@/lib/trpc/routers/deploy/network/utils";
import { zodResolver } from "@hookform/resolvers/zod";
import { Location2, XMark } from "@unkey/icons";
import { toast } from "@unkey/ui";
import { useEffect } from "react";
import { useForm, useWatch } from "react-hook-form";
import { z } from "zod";
import { RegionFlag } from "../../../../components/region-flag";
import { useProjectData } from "../../../data-provider";
import { EditableSettingCard } from "../shared/editable-setting-card";

const regionsSchema = z.object({
  regions: z.array(z.string()).min(1, "Select at least one region"),
});

type RegionsFormValues = z.infer<typeof regionsSchema>;


export const Regions = () => {
  const { environments } = useProjectData();
  const environmentId = environments[0]?.id;

  const { data: settingsData } = trpc.deploy.environmentSettings.get.useQuery(
    { environmentId: environmentId ?? "" },
    { enabled: Boolean(environmentId) },
  );

  const { data: availableRegions } = trpc.deploy.environmentSettings.getAvailableRegions.useQuery(
    undefined,
    { enabled: Boolean(environmentId) },
  );

  const regionConfig = (settingsData?.runtimeSettings?.regionConfig as Record<string, number>) ?? {};
  const defaultRegions = Object.keys(regionConfig);

  if (!environmentId) {
    return null;
  }

  return (
    <RegionsForm
      environmentId={environmentId}
      defaultRegions={defaultRegions}
      availableRegions={availableRegions ?? []}
    />
  );
};

type RegionsFormProps = {
  environmentId: string;
  defaultRegions: string[];
  availableRegions: string[];
};


const RegionsForm: React.FC<RegionsFormProps> = ({
  environmentId,
  defaultRegions,
  availableRegions,
}) => {
  const utils = trpc.useUtils();

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
  }, [defaultRegions.join(",")]);

  const currentRegions = useWatch({ control, name: "regions" });

  const unselectedRegions = availableRegions.filter((r) => !currentRegions.includes(r));

  const updateRuntime = trpc.deploy.environmentSettings.updateRuntime.useMutation({
    onSuccess: () => {
      utils.deploy.environmentSettings.get.invalidate({ environmentId });
    },
    onError: (err) => {
      toast.error("Failed to update regions", { description: err.message });
    },
  });

  const onSubmit = async (values: RegionsFormValues) => {
    await updateRuntime.mutateAsync({
      environmentId,
      regions: values.regions,
    });
  };

  const addRegion = (region: string) => {
    if (region && !currentRegions.includes(region)) {
      setValue("regions", [...currentRegions, region], { shouldValidate: true });
    }
  };

  const removeRegion = (region: string) => {
    setValue(
      "regions",
      currentRegions.filter((r) => r !== region),
      { shouldValidate: true },
    );
  };

  const hasChanges =
    currentRegions.length !== defaultRegions.length ||
    currentRegions.some((r) => !defaultRegions.includes(r));

  const displayValue =
    defaultRegions.length === 0 ? (
      "No regions selected"
    ) : defaultRegions.length <= 2 ? (
      <span className="flex items-center gap-1.5">
        {defaultRegions.map((r, i) => (
          <span key={r} className="flex items-center gap-1.5">
            {i > 0 && <span className="text-gray-6">|</span>}
            <span className="flex items-center gap-1">
              <RegionFlag flagCode={mapRegionToFlag(r)} size="xs" shape="circle" className="[&_img]:size-3" />
              <span className="text-gray-12 font-medium">{r}</span>
            </span>
          </span>
        ))}
      </span>
    ) : (
      <span className="flex items-center gap-1">
        {defaultRegions.map((r) => (
          <RegionFlag key={r} flagCode={mapRegionToFlag(r)} size="xs" shape="circle" />
        ))}
      </span>
    );

  const comboboxOptions: ComboboxOption[] = unselectedRegions.map((region) => ({
    value: region,
    searchValue: region,
    label: (
      <div className="flex items-center gap-2">
        <RegionFlag flagCode={mapRegionToFlag(region)} size="xs" className="[&_img]:size-3" />
        <span className="text-gray-11 text-xs font-mono">{region}</span>
      </div>
    ),
  }));

  return (
    <EditableSettingCard
      icon={<Location2 className="text-gray-12" iconSize="xl-medium" />}
      title="Regions"
      description="Geographic regions where your project will run"
      border="both"
      displayValue={displayValue}
      formId="update-regions-form"
      canSave={isValid && !isSubmitting && hasChanges}
      isSaving={updateRuntime.isLoading || isSubmitting}
    >
      <form id="update-regions-form" onSubmit={handleSubmit(onSubmit)}>
        <FormCombobox
          label="Regions"
          description="Traffic is routed to the nearest selected region"
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
                    <RegionFlag flagCode={mapRegionToFlag(r)} size="xs" shape="circle" className="[&_img]:size-3" />
                    {r}
                    {currentRegions.length > 1 && (
                      <button
                        type="button"
                        onClick={(e) => {
                          e.stopPropagation();
                          removeRegion(r);
                        }}
                        className="p-0.5 hover:bg-grayA-4 rounded text-grayA-9 hover:text-accent-12 transition-colors"
                      >
                        <XMark iconSize="sm-regular" />
                      </button>
                    )}
                  </span>
                ))}
              </div>
            )
          }
          searchPlaceholder="Search regions..."
          emptyMessage={<div className="mt-2">No regions available.</div>}
        />
      </form>
    </EditableSettingCard>
  );
};

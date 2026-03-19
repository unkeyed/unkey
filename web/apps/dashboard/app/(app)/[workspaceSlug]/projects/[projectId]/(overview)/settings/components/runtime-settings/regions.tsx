"use client";

import type { ComboboxOption } from "@/components/ui/combobox";
import { FormCombobox } from "@/components/ui/form-combobox";
import { collection } from "@/lib/collections";
import type { EnvironmentSettings } from "@/lib/collections/deploy/environment-settings";
import { trpc } from "@/lib/trpc/client";
import { mapRegionToFlag } from "@/lib/trpc/routers/deploy/network/utils";
import { zodResolver } from "@hookform/resolvers/zod";
import { Location2, XMark } from "@unkey/icons";
import { useContext, useEffect, useMemo } from "react";
import { useForm, useWatch } from "react-hook-form";
import { z } from "zod";
import { RegionFlag } from "../../../../components/region-flag";
import { EnvironmentContext, useEnvironmentSettings } from "../../environment-provider";
import { useMultiEnvironmentSettings } from "../../hooks/use-multi-environment-settings";
import { EnvironmentSliderSection } from "../shared/environment-slider-section";
import { FormSettingCard, resolveSaveState } from "../shared/form-setting-card";
import { SettingDescription } from "../shared/setting-description";

export const Regions = () => {
  const envContext = useContext(EnvironmentContext);

  if (envContext?.variant === "onboarding") {
    return <RegionsSingle />;
  }

  return <RegionsDual />;
};

const buildRegionComboboxOptions = (
  regions: Array<{ id: string; name: string }>,
): ComboboxOption[] =>
  regions.map((region) => ({
    value: region.name,
    searchValue: region.name,
    label: (
      <div className="flex items-center gap-2">
        <RegionFlag flagCode={mapRegionToFlag(region.name)} size="xs" className="[&_img]:size-3" />
        <span className="text-gray-11 text-xs font-mono">{region.name}</span>
      </div>
    ),
  }));

const RegionTags = ({
  regions,
  onRemove,
  canRemove,
}: {
  regions: string[];
  onRemove: (region: string) => void;
  canRemove: boolean;
}) => (
  <div className="w-full flex flex-wrap gap-1.5 py-0.5">
    {regions.map((r) => (
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
        {canRemove && (
          //biome-ignore lint/a11y/useKeyWithClickEvents: we can't use button here otherwise we'll nest two buttons
          <span
            onClick={(e) => {
              e.stopPropagation();
              onRemove(r);
            }}
            className="p-0.5 hover:bg-grayA-4 rounded text-grayA-9 hover:text-accent-12 transition-colors"
          >
            <XMark iconSize="sm-regular" />
          </span>
        )}
      </span>
    ))}
  </div>
);

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

const regionsSingleSchema = z.object({
  regions: z.array(z.string()).min(1, "Select at least one region"),
});

type RegionsSingleFormValues = z.infer<typeof regionsSingleSchema>;

const RegionsSingle = () => {
  const { settings, variant } = useEnvironmentSettings();
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
  } = useForm<RegionsSingleFormValues>({
    resolver: zodResolver(regionsSingleSchema),
    mode: "onChange",
    defaultValues: { regions: defaultRegions },
  });

  useEffect(() => {
    reset({ regions: defaultRegions });
  }, [defaultRegions, reset]);

  const currentRegions = useWatch({ control, name: "regions" });
  const allRegions = availableRegions ?? [];
  const unselectedRegions = allRegions.filter((r) => !currentRegions.includes(r.name));

  const onSubmit = async (values: RegionsSingleFormValues) => {
    collection.environmentSettings.update(environmentId, (draft) => {
      const defaultReplicas = draft.regions.at(0)?.replicas ?? 1;
      draft.regions = values.regions.map((name) => {
        const existing = draft.regions.find((r) => r.name === name);
        if (existing) {
          return existing;
        }
        const available = (availableRegions ?? []).find((r) => r.name === name);
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
    currentRegions.length !== defaultRegions.length ||
    currentRegions.some((r) => !defaultRegions.includes(r));

  const saveState = resolveSaveState([
    [isSubmitting, { status: "saving" }],
    [!isValid, { status: "disabled" }],
    [!hasChanges, { status: "disabled", reason: "No changes to save" }],
  ]);

  return (
    <FormSettingCard
      icon={<Location2 className="text-gray-12" iconSize="xl-medium" />}
      title="Regions"
      description="Geographic regions where your project will run"
      displayValue={<RegionDisplayValue regions={defaultRegions} />}
      onSubmit={handleSubmit(onSubmit)}
      saveState={saveState}
      autoSave={variant === "onboarding"}
    >
      <FormCombobox
        optional
        className="w-[480px]"
        options={buildRegionComboboxOptions(unselectedRegions)}
        value=""
        onSelect={addRegion}
        closeOnSelect={false}
        placeholder={
          currentRegions.length === 0 ? (
            <span className="text-grayA-8 w-full text-left">Select a region</span>
          ) : (
            <RegionTags
              regions={currentRegions}
              onRemove={removeRegion}
              canRemove={currentRegions.length > 1}
            />
          )
        }
        searchPlaceholder="Search regions..."
        emptyMessage={<div className="mt-2">No regions available.</div>}
      />

      <SettingDescription>
        Traffic is routed to the nearest selected region. Changes apply on next deploy.
      </SettingDescription>
    </FormSettingCard>
  );
};

const regionsDualSchema = z.object({
  productionRegions: z.array(z.string()).min(1, "Select at least one region"),
  previewRegions: z.array(z.string()).min(1, "Select at least one region"),
});

type RegionsDualFormValues = z.infer<typeof regionsDualSchema>;

const RegionsDual = () => {
  const multiSettings = useMultiEnvironmentSettings();

  if (!multiSettings) {
    return null;
  }

  return <RegionsDualInner production={multiSettings.production} preview={multiSettings.preview} />;
};

type RegionsDualInnerProps = {
  production: EnvironmentSettings;
  preview: EnvironmentSettings;
};

const RegionsDualInner = ({ production, preview }: RegionsDualInnerProps) => {
  const defaultProdRegions = useMemo(
    () => production.regions.map((r) => r.name),
    [production.regions],
  );
  const defaultPreviewRegions = useMemo(
    () => preview.regions.map((r) => r.name),
    [preview.regions],
  );

  const { data: availableRegions } = trpc.deploy.environmentSettings.getAvailableRegions.useQuery(
    undefined,
    { enabled: Boolean(production.environmentId) },
  );

  const {
    handleSubmit,
    setValue,
    formState: { isValid, isSubmitting },
    control,
    reset,
  } = useForm<RegionsDualFormValues>({
    resolver: zodResolver(regionsDualSchema),
    mode: "onChange",
    defaultValues: {
      productionRegions: defaultProdRegions,
      previewRegions: defaultPreviewRegions,
    },
  });

  useEffect(() => {
    reset({
      productionRegions: defaultProdRegions,
      previewRegions: defaultPreviewRegions,
    });
  }, [defaultProdRegions, defaultPreviewRegions, reset]);

  const currentProdRegions = useWatch({ control, name: "productionRegions" });
  const currentPreviewRegions = useWatch({ control, name: "previewRegions" });

  const allRegions = availableRegions ?? [];
  const unselectedProdRegions = allRegions.filter((r) => !currentProdRegions.includes(r.name));
  const unselectedPreviewRegions = allRegions.filter(
    (r) => !currentPreviewRegions.includes(r.name),
  );

  const onSubmit = async (values: RegionsDualFormValues) => {
    const prodChanged =
      values.productionRegions.length !== defaultProdRegions.length ||
      values.productionRegions.some((r) => !defaultProdRegions.includes(r));

    const prevChanged =
      values.previewRegions.length !== defaultPreviewRegions.length ||
      values.previewRegions.some((r) => !defaultPreviewRegions.includes(r));

    if (prodChanged) {
      collection.environmentSettings.update(production.environmentId, (draft) => {
        const defaultReplicas = draft.regions[0]?.replicas ?? 1;
        draft.regions = values.productionRegions.map((name) => {
          const existing = draft.regions.find((r) => r.name === name);
          if (existing) {
            return existing;
          }
          const available = (availableRegions ?? []).find((r) => r.name === name);
          return { id: available?.id ?? name, name, replicas: defaultReplicas };
        });
      });
    }

    if (prevChanged) {
      collection.environmentSettings.update(preview.environmentId, (draft) => {
        const defaultReplicas = draft.regions[0]?.replicas ?? 1;
        draft.regions = values.previewRegions.map((name) => {
          const existing = draft.regions.find((r) => r.name === name);
          if (existing) {
            return existing;
          }
          const available = (availableRegions ?? []).find((r) => r.name === name);
          return { id: available?.id ?? name, name, replicas: defaultReplicas };
        });
      });
    }
  };

  const prodHasChanges =
    currentProdRegions.length !== defaultProdRegions.length ||
    currentProdRegions.some((r) => !defaultProdRegions.includes(r));
  const previewHasChanges =
    currentPreviewRegions.length !== defaultPreviewRegions.length ||
    currentPreviewRegions.some((r) => !defaultPreviewRegions.includes(r));
  const hasChanges = prodHasChanges || previewHasChanges;

  const saveState = resolveSaveState([
    [isSubmitting, { status: "saving" }],
    [!isValid, { status: "disabled" }],
    [!hasChanges, { status: "disabled", reason: "No changes to save" }],
  ]);

  const addProdRegion = (region: string) => {
    if (region && !currentProdRegions.includes(region)) {
      setValue("productionRegions", [...currentProdRegions, region], { shouldValidate: true });
    }
  };

  const removeProdRegion = (region: string) => {
    setValue(
      "productionRegions",
      currentProdRegions.filter((r) => r !== region),
      { shouldValidate: true },
    );
  };

  const addPreviewRegion = (region: string) => {
    if (region && !currentPreviewRegions.includes(region)) {
      setValue("previewRegions", [...currentPreviewRegions, region], { shouldValidate: true });
    }
  };

  const removePreviewRegion = (region: string) => {
    setValue(
      "previewRegions",
      currentPreviewRegions.filter((r) => r !== region),
      { shouldValidate: true },
    );
  };

  return (
    <FormSettingCard
      icon={<Location2 className="text-gray-12" iconSize="xl-medium" />}
      title="Regions"
      description="Geographic regions where your project will run"
      displayValue={
        <div className="flex items-center gap-3">
          <EnvironmentDisplayValue label="Production" regions={defaultProdRegions} />
          <span className="text-gray-8">|</span>
          <EnvironmentDisplayValue label="Preview" regions={defaultPreviewRegions} />
        </div>
      }
      onSubmit={handleSubmit(onSubmit)}
      saveState={saveState}
    >
      <EnvironmentSliderSection label="Production">
        <FormCombobox
          className="w-[480px]"
          options={buildRegionComboboxOptions(unselectedProdRegions)}
          value=""
          onSelect={addProdRegion}
          closeOnSelect={false}
          placeholder={
            currentProdRegions.length === 0 ? (
              <span className="text-grayA-8 w-full text-left">Select a region</span>
            ) : (
              <RegionTags
                regions={currentProdRegions}
                onRemove={removeProdRegion}
                canRemove={currentProdRegions.length > 1}
              />
            )
          }
          searchPlaceholder="Search regions..."
          emptyMessage={<div className="mt-2">No regions available.</div>}
        />
      </EnvironmentSliderSection>

      <EnvironmentSliderSection label="Preview">
        <FormCombobox
          className="w-[480px]"
          options={buildRegionComboboxOptions(unselectedPreviewRegions)}
          value=""
          onSelect={addPreviewRegion}
          closeOnSelect={false}
          placeholder={
            currentPreviewRegions.length === 0 ? (
              <span className="text-grayA-8 w-full text-left">Select a region</span>
            ) : (
              <RegionTags
                regions={currentPreviewRegions}
                onRemove={removePreviewRegion}
                canRemove={currentPreviewRegions.length > 1}
              />
            )
          }
          searchPlaceholder="Search regions..."
          emptyMessage={<div className="mt-2">No regions available.</div>}
        />
      </EnvironmentSliderSection>

      <SettingDescription>
        Traffic is routed to the nearest selected region. Changes apply on next deploy.
      </SettingDescription>
    </FormSettingCard>
  );
};

const EnvironmentDisplayValue = ({ label, regions }: { label: string; regions: string[] }) => (
  <div className="flex items-center gap-1.5">
    <span className="text-gray-11 text-xs font-normal">{label}</span>
    {regions.map((r) => (
      <RegionFlag
        key={r}
        flagCode={mapRegionToFlag(r)}
        size="xs"
        shape="circle"
        className="[&_img]:size-3"
      />
    ))}
  </div>
);

"use client";

import { collection } from "@/lib/collections";
import type { EnvironmentSettings } from "@/lib/collections/deploy/environment-settings";
import { mapRegionToFlag } from "@/lib/trpc/routers/deploy/network/utils";
import { zodResolver } from "@hookform/resolvers/zod";
import { Connections3 } from "@unkey/icons";
import { Slider } from "@unkey/ui";
import { useContext, useEffect } from "react";
import { useForm, useWatch } from "react-hook-form";
import { z } from "zod";
import { RegionFlag } from "../../../../components/region-flag";
import { EnvironmentContext, useEnvironmentSettings } from "../../environment-provider";
import { useMultiEnvironmentSettings } from "../../hooks/use-multi-environment-settings";
import { useUpdateAllEnvironments } from "../../hooks/use-update-all-environments";
import { SettingDescription, WideContent } from "../shared/form-blocks";
import { FormSettingCard, type SaveState, resolveSaveState } from "../shared/form-setting-card";
import { EnvironmentDisplayValue } from "../shared/resource-slider/environment-display-value";
import { EnvironmentSliderSection } from "../shared/resource-slider/environment-slider-section";

const REPLICAS_MIN = 1;
const REPLICAS_MAX = 4;
const COLOR_VAR = "featureA";

const formatRangeParts = (replicasMin: number, replicasMax: number) => ({
  value: replicasMin === replicasMax ? String(replicasMax) : `${replicasMin} – ${replicasMax}`,
  unit: "",
});

const rangeSchema = z
  .object({
    replicasMin: z.number().int().min(REPLICAS_MIN).max(REPLICAS_MAX),
    replicasMax: z.number().int().min(REPLICAS_MIN).max(REPLICAS_MAX),
  })
  .refine((d) => d.replicasMin <= d.replicasMax, {
    message: "replicasMin must be ≤ replicasMax",
    path: ["replicasMin"],
  });

type RangeFormValues = z.infer<typeof rangeSchema>;

const buildSliderRangeStyle = (replicasMin: number, replicasMax: number) => {
  const span = REPLICAS_MAX - REPLICAS_MIN;
  const left = span > 0 ? (replicasMin - REPLICAS_MIN) / span : 0;
  const right = span > 0 ? (replicasMax - REPLICAS_MIN) / span : 0;
  return {
    background: `linear-gradient(to right, hsla(var(--${COLOR_VAR}-4)), hsla(var(--${COLOR_VAR}-12)))`,
    backgroundSize: `${right > left ? 100 / (right - left) : 10000}% 100%`,
    backgroundPosition: `${left > 0 ? (100 * left) / (1 - left) : 0}% 0`,
    backgroundRepeat: "no-repeat",
  };
};

const RegionFlags = ({ settings }: { settings: EnvironmentSettings }) => {
  const regions = settings.regions.map((r) => r.name);
  if (regions.length === 0) {
    return null;
  }
  return (
    <div className="flex items-center gap-1.5">
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
};

const noRegionsCheck = (settings: EnvironmentSettings[]): SaveState | null => {
  const anyHasRegions = settings.some((s) => s.regions.length > 0);
  if (!anyHasRegions) {
    return {
      status: "disabled",
      reason: "Select at least one region before setting instance count",
    };
  }
  return null;
};

const readRange = (s: EnvironmentSettings): RangeFormValues => ({
  replicasMin: s.regions[0]?.replicasMin ?? 1,
  replicasMax: s.regions[0]?.replicasMax ?? 1,
});

const writeRange = (draft: EnvironmentSettings, values: RangeFormValues) => {
  for (const region of draft.regions) {
    region.replicasMin = values.replicasMin;
    region.replicasMax = values.replicasMax;
  }
};

export const Instances = () => {
  const envContext = useContext(EnvironmentContext);

  if (!envContext) {
    throw new Error("Instances must be used within EnvironmentProvider");
  }

  if (envContext.variant === "onboarding") {
    return <SingleMode />;
  }

  return <DualMode />;
};

const description =
  "Autoscaling range per region. Scales up to the maximum based on CPU usage, down to the minimum when load is low.";
const settingDescription =
  "Changes apply on next deploy. During beta, instances are limited to 4 per region. Contact support@unkey.com if you need more.";

const SingleMode = () => {
  const { settings, variant } = useEnvironmentSettings();
  const updateAllEnvironments = useUpdateAllEnvironments();
  const defaultValues = readRange(settings);

  const {
    handleSubmit,
    setValue,
    formState: { isValid, isSubmitting },
    control,
    reset,
  } = useForm<RangeFormValues>({
    resolver: zodResolver(rangeSchema),
    mode: "onChange",
    defaultValues,
  });

  useEffect(() => {
    reset({ replicasMin: defaultValues.replicasMin, replicasMax: defaultValues.replicasMax });
  }, [defaultValues.replicasMin, defaultValues.replicasMax, reset]);

  const currentReplicasMin = useWatch({ control, name: "replicasMin" });
  const currentReplicasMax = useWatch({ control, name: "replicasMax" });

  const onSubmit = async (values: RangeFormValues) => {
    updateAllEnvironments((draft) => {
      writeRange(draft, values);
    });
  };

  const hasChanges =
    currentReplicasMin !== defaultValues.replicasMin ||
    currentReplicasMax !== defaultValues.replicasMax;
  const extraCheck = noRegionsCheck([settings]);
  const saveState = resolveSaveState([
    ...(extraCheck ? [[true, extraCheck] as [boolean, SaveState]] : []),
    [isSubmitting, { status: "saving" }],
    [!isValid, { status: "disabled" }],
    [!hasChanges, { status: "disabled", reason: "No changes to save" }],
  ]);

  const displayParts = formatRangeParts(defaultValues.replicasMin, defaultValues.replicasMax);

  return (
    <FormSettingCard
      icon={<Connections3 className="text-gray-12" iconSize="xl-medium" />}
      title="Instances"
      description={description}
      displayValue={<span className="font-medium text-gray-12">{displayParts.value}</span>}
      onSubmit={handleSubmit(onSubmit)}
      saveState={saveState}
      autoSave
    >
      <WideContent>
        <div className="flex items-center gap-3">
          <Slider
            min={REPLICAS_MIN}
            max={REPLICAS_MAX}
            step={1}
            value={[currentReplicasMin, currentReplicasMax]}
            onValueChange={([replicasMin, replicasMax]) => {
              if (replicasMin !== undefined) {
                setValue("replicasMin", replicasMin, { shouldValidate: true });
              }
              if (replicasMax !== undefined) {
                setValue("replicasMax", replicasMax, { shouldValidate: true });
              }
            }}
            onValueCommit={
              variant === "onboarding"
                ? ([replicasMin, replicasMax]) => {
                    if (replicasMin === undefined || replicasMax === undefined) {
                      return;
                    }
                    if (
                      replicasMin === defaultValues.replicasMin &&
                      replicasMax === defaultValues.replicasMax
                    ) {
                      return;
                    }
                    updateAllEnvironments((draft) => {
                      writeRange(draft, { replicasMin, replicasMax });
                    });
                  }
                : undefined
            }
            className="flex-1 max-w-(--setting-w)"
            rangeStyle={buildSliderRangeStyle(currentReplicasMin, currentReplicasMax)}
          />
          <RegionFlags settings={settings} />
          <span className="text-[13px] font-medium text-gray-12">
            {formatRangeParts(currentReplicasMin, currentReplicasMax).value}
          </span>
        </div>
        <SettingDescription>{settingDescription}</SettingDescription>
      </WideContent>
    </FormSettingCard>
  );
};

const dualSchema = z
  .object({
    productionReplicasMin: z.number().int().min(REPLICAS_MIN).max(REPLICAS_MAX),
    productionReplicasMax: z.number().int().min(REPLICAS_MIN).max(REPLICAS_MAX),
    previewReplicasMin: z.number().int().min(REPLICAS_MIN).max(REPLICAS_MAX),
    previewReplicasMax: z.number().int().min(REPLICAS_MIN).max(REPLICAS_MAX),
  })
  .refine((d) => d.productionReplicasMin <= d.productionReplicasMax, {
    message: "replicasMin must be ≤ replicasMax",
    path: ["productionReplicasMin"],
  })
  .refine((d) => d.previewReplicasMin <= d.previewReplicasMax, {
    message: "replicasMin must be ≤ replicasMax",
    path: ["previewReplicasMin"],
  });

type DualFormValues = z.infer<typeof dualSchema>;

const DualMode = () => {
  const multiSettings = useMultiEnvironmentSettings();

  if (!multiSettings) {
    return null;
  }

  return <DualInner production={multiSettings.production} preview={multiSettings.preview} />;
};

type DualInnerProps = {
  production: EnvironmentSettings;
  preview: EnvironmentSettings;
};

const DualInner = ({ production, preview }: DualInnerProps) => {
  const defaultProduction = readRange(production);
  const defaultPreview = readRange(preview);

  const {
    handleSubmit,
    setValue,
    formState: { isValid, isSubmitting },
    control,
    reset,
  } = useForm<DualFormValues>({
    resolver: zodResolver(dualSchema),
    mode: "onChange",
    defaultValues: {
      productionReplicasMin: defaultProduction.replicasMin,
      productionReplicasMax: defaultProduction.replicasMax,
      previewReplicasMin: defaultPreview.replicasMin,
      previewReplicasMax: defaultPreview.replicasMax,
    },
  });

  useEffect(() => {
    reset({
      productionReplicasMin: defaultProduction.replicasMin,
      productionReplicasMax: defaultProduction.replicasMax,
      previewReplicasMin: defaultPreview.replicasMin,
      previewReplicasMax: defaultPreview.replicasMax,
    });
  }, [
    defaultProduction.replicasMin,
    defaultProduction.replicasMax,
    defaultPreview.replicasMin,
    defaultPreview.replicasMax,
    reset,
  ]);

  const currentProductionReplicasMin = useWatch({ control, name: "productionReplicasMin" });
  const currentProductionReplicasMax = useWatch({ control, name: "productionReplicasMax" });
  const currentPreviewReplicasMin = useWatch({ control, name: "previewReplicasMin" });
  const currentPreviewReplicasMax = useWatch({ control, name: "previewReplicasMax" });

  const onSubmit = async (values: DualFormValues) => {
    if (
      values.productionReplicasMin !== defaultProduction.replicasMin ||
      values.productionReplicasMax !== defaultProduction.replicasMax
    ) {
      collection.environmentSettings.update(production.environmentId, (draft) => {
        writeRange(draft, {
          replicasMin: values.productionReplicasMin,
          replicasMax: values.productionReplicasMax,
        });
      });
    }
    if (
      values.previewReplicasMin !== defaultPreview.replicasMin ||
      values.previewReplicasMax !== defaultPreview.replicasMax
    ) {
      collection.environmentSettings.update(preview.environmentId, (draft) => {
        writeRange(draft, {
          replicasMin: values.previewReplicasMin,
          replicasMax: values.previewReplicasMax,
        });
      });
    }
  };

  const productionHasChanges =
    currentProductionReplicasMin !== defaultProduction.replicasMin ||
    currentProductionReplicasMax !== defaultProduction.replicasMax;
  const previewHasChanges =
    currentPreviewReplicasMin !== defaultPreview.replicasMin ||
    currentPreviewReplicasMax !== defaultPreview.replicasMax;
  const hasChanges = productionHasChanges || previewHasChanges;

  const extraCheck = noRegionsCheck([production, preview]);
  const saveState = resolveSaveState([
    ...(extraCheck ? [[true, extraCheck] as [boolean, SaveState]] : []),
    [isSubmitting, { status: "saving" }],
    [!isValid, { status: "disabled" }],
    [!hasChanges, { status: "disabled", reason: "No changes to save" }],
  ]);

  return (
    <FormSettingCard
      icon={<Connections3 className="text-gray-12" iconSize="xl-medium" />}
      title="Instances"
      description={description}
      displayValue={
        <div className="flex items-center gap-3">
          <EnvironmentDisplayValue
            label="Production"
            parts={formatRangeParts(defaultProduction.replicasMin, defaultProduction.replicasMax)}
          />
          <span className="text-gray-8">|</span>
          <EnvironmentDisplayValue
            label="Preview"
            parts={formatRangeParts(defaultPreview.replicasMin, defaultPreview.replicasMax)}
          />
        </div>
      }
      onSubmit={handleSubmit(onSubmit)}
      saveState={saveState}
    >
      <WideContent>
        <DualSliderSection
          label="Production"
          settings={production}
          replicasMin={currentProductionReplicasMin}
          replicasMax={currentProductionReplicasMax}
          onChange={(replicasMin, replicasMax) => {
            setValue("productionReplicasMin", replicasMin, { shouldValidate: true });
            setValue("productionReplicasMax", replicasMax, { shouldValidate: true });
          }}
        />
        <DualSliderSection
          label="Preview"
          settings={preview}
          replicasMin={currentPreviewReplicasMin}
          replicasMax={currentPreviewReplicasMax}
          onChange={(replicasMin, replicasMax) => {
            setValue("previewReplicasMin", replicasMin, { shouldValidate: true });
            setValue("previewReplicasMax", replicasMax, { shouldValidate: true });
          }}
        />
        <SettingDescription>{settingDescription}</SettingDescription>
      </WideContent>
    </FormSettingCard>
  );
};

type DualSliderSectionProps = {
  label: string;
  settings: EnvironmentSettings;
  replicasMin: number;
  replicasMax: number;
  onChange: (replicasMin: number, replicasMax: number) => void;
};

const DualSliderSection = ({
  label,
  settings,
  replicasMin,
  replicasMax,
  onChange,
}: DualSliderSectionProps) => (
  <EnvironmentSliderSection label={label}>
    <div className="flex items-center gap-3">
      <Slider
        min={REPLICAS_MIN}
        max={REPLICAS_MAX}
        step={1}
        value={[replicasMin, replicasMax]}
        onValueChange={([nextReplicasMin, nextReplicasMax]) => {
          if (nextReplicasMin !== undefined && nextReplicasMax !== undefined) {
            onChange(nextReplicasMin, nextReplicasMax);
          }
        }}
        className="flex-1 max-w-(--setting-w)"
        rangeStyle={buildSliderRangeStyle(replicasMin, replicasMax)}
      />
      <RegionFlags settings={settings} />
      <span className="text-[13px] font-medium text-gray-12">
        {formatRangeParts(replicasMin, replicasMax).value}
      </span>
    </div>
  </EnvironmentSliderSection>
);

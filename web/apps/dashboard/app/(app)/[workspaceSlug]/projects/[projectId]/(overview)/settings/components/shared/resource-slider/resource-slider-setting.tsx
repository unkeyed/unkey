"use client";

import { collection } from "@/lib/collections";
import type { EnvironmentSettings } from "@/lib/collections/deploy/environment-settings";
import type { FormattedParts } from "@/lib/utils/deployment-formatters";
import { zodResolver } from "@hookform/resolvers/zod";
import { Slider } from "@unkey/ui";
import type React from "react";
import { useContext, useEffect, useMemo } from "react";
import { useForm, useWatch } from "react-hook-form";
import { z } from "zod";
import { EnvironmentContext, useEnvironmentSettings } from "../../../environment-provider";
import { useMultiEnvironmentSettings } from "../../../hooks/use-multi-environment-settings";
import { useUpdateAllEnvironments } from "../../../hooks/use-update-all-environments";
import { SettingDescription, WideContent } from "../form-blocks";
import { FormSettingCard, type SaveState, resolveSaveState } from "../form-setting-card";
import { EnvironmentDisplayValue } from "./environment-display-value";
import { EnvironmentSliderSection } from "./environment-slider-section";
import { buildSliderRangeStyle, indexToValue, valueToIndex } from "./slider-utils";

type SliderStrategy =
  | {
      kind: "index-mapped";
      options: readonly { readonly label: string; readonly value: number }[];
      fallback: number;
    }
  | { kind: "direct"; min: number; max: number; step: number };

export type ResourceSliderConfig = {
  icon: React.ReactNode;
  title: string;
  description: string;
  settingDescription: string;
  colorVar: string;
  slider: SliderStrategy;
  formatValue: (n: number) => FormattedParts;
  readValue: (s: EnvironmentSettings) => number;
  writeValue: (draft: EnvironmentSettings, value: number) => void;
  extraSaveChecks?: (settings: EnvironmentSettings[]) => SaveState | null;
  sliderAdornment?: (s: EnvironmentSettings) => React.ReactNode;
};

function getSliderProps(strategy: SliderStrategy, currentValue: number) {
  if (strategy.kind === "index-mapped") {
    const index = valueToIndex(strategy.options, currentValue);
    return {
      min: 0,
      max: strategy.options.length - 1,
      step: 1,
      sliderValue: index,
      toFormValue: (v: number) => indexToValue(strategy.options, v, strategy.fallback),
      rangeIndex: index,
      rangeMin: 0,
      rangeMax: strategy.options.length - 1,
    };
  }
  return {
    min: strategy.min,
    max: strategy.max,
    step: strategy.step,
    sliderValue: currentValue,
    toFormValue: (v: number) => v,
    rangeIndex: currentValue,
    rangeMin: strategy.min,
    rangeMax: strategy.max,
  };
}

export const ResourceSliderSetting = ({ config }: { config: ResourceSliderConfig }) => {
  const envContext = useContext(EnvironmentContext);

  if (!envContext) {
    throw new Error("ResourceSliderSetting must be used within EnvironmentProvider");
  }

  if (envContext.variant === "onboarding") {
    return <SingleMode config={config} />;
  }

  return <DualMode config={config} />;
};

const singleSchema = z.object({ value: z.number() });
type SingleFormValues = z.infer<typeof singleSchema>;

const SingleMode = ({ config }: { config: ResourceSliderConfig }) => {
  const { settings, variant } = useEnvironmentSettings();
  const updateAllEnvironments = useUpdateAllEnvironments();
  const defaultValue = config.readValue(settings);

  const {
    handleSubmit,
    setValue,
    formState: { isValid, isSubmitting },
    control,
    reset,
  } = useForm<SingleFormValues>({
    resolver: zodResolver(singleSchema),
    mode: "onChange",
    defaultValues: { value: defaultValue },
  });

  useEffect(() => {
    reset({ value: defaultValue });
  }, [defaultValue, reset]);

  const currentValue = useWatch({ control, name: "value" });

  const onSubmit = async (values: SingleFormValues) => {
    updateAllEnvironments((draft) => {
      config.writeValue(draft, values.value);
    });
  };

  const hasChanges = currentValue !== defaultValue;
  const sp = getSliderProps(config.slider, currentValue);

  const extraCheck = config.extraSaveChecks?.([settings]);
  const saveState = resolveSaveState([
    ...(extraCheck ? [[true, extraCheck] as [boolean, SaveState]] : []),
    [isSubmitting, { status: "saving" }],
    [!isValid, { status: "disabled" }],
    [!hasChanges, { status: "disabled", reason: "No changes to save" }],
  ]);

  const displayParts = config.formatValue(defaultValue);

  return (
    <FormSettingCard
      icon={config.icon}
      title={config.title}
      description={config.description}
      displayValue={
        <span>
          <span className="font-medium text-gray-12">{displayParts.value}</span>{" "}
          <span className="text-gray-11">{displayParts.unit}</span>
        </span>
      }
      onSubmit={handleSubmit(onSubmit)}
      saveState={saveState}
      autoSave
    >
      <WideContent>
        <div className="flex items-center gap-3">
          <Slider
            min={sp.min}
            max={sp.max}
            step={sp.step}
            value={[sp.sliderValue]}
            onValueChange={([v]) => {
              if (v !== undefined) {
                setValue("value", sp.toFormValue(v), { shouldValidate: true });
              }
            }}
            onValueCommit={
              variant === "onboarding"
                ? ([v]) => {
                    if (v !== undefined) {
                      const newValue = sp.toFormValue(v);
                      if (newValue !== defaultValue) {
                        updateAllEnvironments((draft) => {
                          config.writeValue(draft, newValue);
                        });
                      }
                    }
                  }
                : undefined
            }
            className="flex-1 max-w-(--setting-w)"
            rangeStyle={buildSliderRangeStyle(
              sp.rangeIndex,
              sp.rangeMax,
              sp.rangeMin,
              config.colorVar,
            )}
          />
          {config.sliderAdornment?.(settings)}
          <span className="text-[13px]">
            <span className="font-medium text-gray-12">
              {config.formatValue(currentValue).value}
            </span>{" "}
            <span className="text-gray-11">{config.formatValue(currentValue).unit}</span>
          </span>
        </div>
        <SettingDescription>{config.settingDescription}</SettingDescription>
      </WideContent>
    </FormSettingCard>
  );
};

const dualSchema = z.object({ production: z.number(), preview: z.number() });
type DualFormValues = z.infer<typeof dualSchema>;

const DualMode = ({ config }: { config: ResourceSliderConfig }) => {
  const multiSettings = useMultiEnvironmentSettings();

  if (!multiSettings) {
    return null;
  }

  return (
    <DualInner
      config={config}
      production={multiSettings.production}
      preview={multiSettings.preview}
    />
  );
};

type DualInnerProps = {
  config: ResourceSliderConfig;
  production: EnvironmentSettings;
  preview: EnvironmentSettings;
};

const DualInner = ({ config, production, preview }: DualInnerProps) => {
  const defaultProd = config.readValue(production);
  const defaultPreview = config.readValue(preview);

  const {
    handleSubmit,
    setValue,
    formState: { isValid, isSubmitting },
    control,
    reset,
  } = useForm<DualFormValues>({
    resolver: zodResolver(dualSchema),
    mode: "onChange",
    defaultValues: { production: defaultProd, preview: defaultPreview },
  });

  useEffect(() => {
    reset({ production: defaultProd, preview: defaultPreview });
  }, [defaultProd, defaultPreview, reset]);

  const currentProd = useWatch({ control, name: "production" });
  const currentPreview = useWatch({ control, name: "preview" });

  const onSubmit = async (values: DualFormValues) => {
    if (values.production !== defaultProd) {
      collection.environmentSettings.update(production.environmentId, (draft) => {
        config.writeValue(draft, values.production);
      });
    }
    if (values.preview !== defaultPreview) {
      collection.environmentSettings.update(preview.environmentId, (draft) => {
        config.writeValue(draft, values.preview);
      });
    }
  };

  const hasChanges = currentProd !== defaultProd || currentPreview !== defaultPreview;

  const extraCheck = config.extraSaveChecks?.([production, preview]);
  const saveState = resolveSaveState([
    ...(extraCheck ? [[true, extraCheck] as [boolean, SaveState]] : []),
    [isSubmitting, { status: "saving" }],
    [!isValid, { status: "disabled" }],
    [!hasChanges, { status: "disabled", reason: "No changes to save" }],
  ]);

  const prodSp = useMemo(
    () => getSliderProps(config.slider, currentProd),
    [config.slider, currentProd],
  );
  const previewSp = useMemo(
    () => getSliderProps(config.slider, currentPreview),
    [config.slider, currentPreview],
  );

  return (
    <FormSettingCard
      icon={config.icon}
      title={config.title}
      description={config.description}
      displayValue={
        <div className="flex items-center gap-3">
          <EnvironmentDisplayValue label="Production" parts={config.formatValue(defaultProd)} />
          <span className="text-gray-8">|</span>
          <EnvironmentDisplayValue label="Preview" parts={config.formatValue(defaultPreview)} />
        </div>
      }
      onSubmit={handleSubmit(onSubmit)}
      saveState={saveState}
    >
      <WideContent>
        <DualSliderSection
          label="Production"
          config={config}
          sp={prodSp}
          settings={production}
          onSliderChange={(v) => setValue("production", v, { shouldValidate: true })}
        />

        <DualSliderSection
          label="Preview"
          config={config}
          sp={previewSp}
          settings={preview}
          onSliderChange={(v) => setValue("preview", v, { shouldValidate: true })}
        />

        <SettingDescription>{config.settingDescription}</SettingDescription>
      </WideContent>
    </FormSettingCard>
  );
};

type SliderSectionProps = {
  label: string;
  config: ResourceSliderConfig;
  sp: ReturnType<typeof getSliderProps>;
  settings: EnvironmentSettings;
  onSliderChange: (value: number) => void;
};

const DualSliderSection = ({ label, config, sp, settings, onSliderChange }: SliderSectionProps) => (
  <EnvironmentSliderSection label={label}>
    <div className="flex items-center gap-3">
      <Slider
        min={sp.min}
        max={sp.max}
        step={sp.step}
        value={[sp.sliderValue]}
        onValueChange={([v]) => {
          if (v !== undefined) {
            onSliderChange(sp.toFormValue(v));
          }
        }}
        className="flex-1 max-w-[var(--setting-w)]"
        rangeStyle={buildSliderRangeStyle(sp.rangeIndex, sp.rangeMax, sp.rangeMin, config.colorVar)}
      />
      {config.sliderAdornment?.(settings)}
      <span className="text-[13px]">
        <span className="font-medium text-gray-12">
          {config.formatValue(sp.toFormValue(sp.sliderValue)).value}
        </span>{" "}
        <span className="text-gray-11">
          {config.formatValue(sp.toFormValue(sp.sliderValue)).unit}
        </span>
      </span>
    </div>
  </EnvironmentSliderSection>
);

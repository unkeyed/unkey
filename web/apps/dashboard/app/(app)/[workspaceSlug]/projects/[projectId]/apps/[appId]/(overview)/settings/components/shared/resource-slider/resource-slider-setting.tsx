"use client";

import { collection } from "@/lib/collections";
import type { EnvironmentSettings } from "@/lib/collections/deploy/environment-settings";
import { freeTierQuotas } from "@/lib/quotas";
import type { FormattedParts } from "@/lib/utils/deployment-formatters";
import { useWorkspace } from "@/providers/workspace-provider";
import { zodResolver } from "@hookform/resolvers/zod";
import type { Quotas } from "@unkey/db";
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
  /**
   * Returns the per-instance quota that caps this resource. Index-mapped sliders
   * hide options above it, so the slider matches the workspace's real quota.
   * Falls back to the default quota until quotas load.
   */
  resolveMax?: (quotas: Quotas | null) => number;
};

// Numeric quota columns a slider can bound to. Taken from freeTierQuotas so the
// fallback lookup always resolves; that type omits pk and workspaceId.
type PerInstanceQuotaKey = {
  [K in keyof typeof freeTierQuotas]: (typeof freeTierQuotas)[K] extends number ? K : never;
}[keyof typeof freeTierQuotas];

function optionForValue(value: number, formatValue: (n: number) => FormattedParts) {
  const parts = formatValue(value);
  return { value, label: parts.unit ? `${parts.value} ${parts.unit}` : parts.value };
}

/**
 * Bounds an index-mapped slider to the workspace quota. Drops options above the
 * cap. When the cap is not already one of the options, adds it as the final stop
 * so a raised quota is always reachable.
 */
function resolveStrategy(
  strategy: SliderStrategy,
  quotas: Quotas | null,
  resolveMax: ResourceSliderConfig["resolveMax"],
  formatValue: (n: number) => FormattedParts,
): SliderStrategy {
  if (strategy.kind !== "index-mapped" || !resolveMax) {
    return strategy;
  }
  const max = resolveMax(quotas);
  const withinQuota = strategy.options.filter((o) => o.value <= max);

  // Quota sits below the lowest defined tier: the only coherent stop is the
  // quota itself. Keeping the smallest tier here would offer a value above the
  // cap and, once the quota is appended, produce an out-of-order list like
  // [{250}, {100}]. Fall back to the smallest tier only for a non-positive cap,
  // which can't happen for real quotas but keeps the option list non-empty.
  if (withinQuota.length === 0) {
    return {
      ...strategy,
      options: max > 0 ? [optionForValue(max, formatValue)] : strategy.options.slice(0, 1),
    };
  }

  const options = [...withinQuota];
  const top = options.at(-1);
  if (top && max > 0 && top.value !== max) {
    options.push(optionForValue(max, formatValue));
  }
  return { ...strategy, options };
}

/**
 * Keeps stored values selectable. resolveStrategy rebuilds the option list from
 * the quota alone, so a value saved under an old quota (e.g. 3000 saved when the
 * cap was 3000, then raised to 5000) can fall off the list. valueToIndex would
 * then return 0 and the thumb would render at the minimum while the label still
 * shows the true value; the next drag auto-saves a silent downgrade. Re-inserting
 * absent stored values and sorting ascending makes the thumb reflect what is
 * stored and leaves any still-valid value selectable.
 */
function ensureValuesSelectable(
  strategy: SliderStrategy,
  values: number[],
  formatValue: (n: number) => FormattedParts,
): SliderStrategy {
  if (strategy.kind !== "index-mapped") {
    return strategy;
  }
  const present = new Set(strategy.options.map((o) => o.value));
  const missing = [...new Set(values.filter((v) => v > 0 && !present.has(v)))];
  if (missing.length === 0) {
    return strategy;
  }
  const options = [...strategy.options, ...missing.map((v) => optionForValue(v, formatValue))];
  options.sort((a, b) => a.value - b.value);
  return { ...strategy, options };
}

type ResourceSliderDefinition = {
  icon: React.ReactNode;
  title: string;
  description: string;
  settingDescription: string;
  colorVar: string;
  options: readonly { readonly label: string; readonly value: number }[];
  fallback: number;
  formatValue: (n: number) => FormattedParts;
  read: (s: EnvironmentSettings) => number;
  write: (draft: EnvironmentSettings, value: number) => void;
  /**
   * Quota column that caps this resource. The slider tops out at the workspace's
   * value for this column, or the default quota until quotas load.
   */
  quotaKey: PerInstanceQuotaKey;
};

/**
 * Builds a quota-bounded, index-mapped slider config. Callers pass the options,
 * the quota column, and how to read and write the value. The slider strategy and
 * quota fallback stay here.
 */
export function defineResourceSlider(definition: ResourceSliderDefinition): ResourceSliderConfig {
  return {
    icon: definition.icon,
    title: definition.title,
    description: definition.description,
    settingDescription: definition.settingDescription,
    colorVar: definition.colorVar,
    slider: { kind: "index-mapped", options: definition.options, fallback: definition.fallback },
    formatValue: definition.formatValue,
    readValue: definition.read,
    writeValue: definition.write,
    resolveMax: (quotas) => quotas?.[definition.quotaKey] ?? freeTierQuotas[definition.quotaKey],
  };
}

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
  const { quotas } = useWorkspace();

  const effectiveConfig = useMemo<ResourceSliderConfig>(
    () => ({
      ...config,
      slider: resolveStrategy(config.slider, quotas, config.resolveMax, config.formatValue),
    }),
    [config, quotas],
  );

  if (!envContext) {
    throw new Error("ResourceSliderSetting must be used within EnvironmentProvider");
  }

  if (envContext.variant === "onboarding") {
    return <SingleMode config={effectiveConfig} />;
  }

  return <DualMode config={effectiveConfig} />;
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

  const slider = useMemo(
    () => ensureValuesSelectable(config.slider, [defaultValue], config.formatValue),
    [config.slider, config.formatValue, defaultValue],
  );

  const hasChanges = currentValue !== defaultValue;
  const sp = getSliderProps(slider, currentValue);

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
            onValueCommitted={
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

  const slider = useMemo(
    () => ensureValuesSelectable(config.slider, [defaultProd, defaultPreview], config.formatValue),
    [config.slider, config.formatValue, defaultProd, defaultPreview],
  );

  const prodSp = useMemo(() => getSliderProps(slider, currentProd), [slider, currentProd]);
  const previewSp = useMemo(() => getSliderProps(slider, currentPreview), [slider, currentPreview]);

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
        className="flex-1 max-w-(--setting-w)"
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

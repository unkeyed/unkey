"use client";

import { rampColorVar } from "@/components/charts/chart-colors";
import type { EnvironmentSettings } from "@/lib/collections/deploy/environment-settings";
import { freeTierLimits } from "@/lib/limits";
import { mapRegionToFlag } from "@/lib/trpc/routers/deploy/network/utils";
import { useWorkspace } from "@/providers/workspace-provider";
import { zodResolver } from "@hookform/resolvers/zod";
import { type SaveState, Slider } from "@unkey/ui";
import { useEffect, useMemo } from "react";
import { useForm, useWatch } from "react-hook-form";
import { z } from "zod";
import { RegionFlag } from "../../../../components/region-flag";
import { useEnvironmentSettings } from "../../environment-provider";
import { useUpdateEnvironment } from "../../hooks/use-update-environment";
import { WideContent } from "../shared/form-blocks";
import { FormSettingCard, resolveSaveState } from "../shared/form-setting-card";

const REPLICAS_MIN = 1;
const COLOR_VAR = "featureA";

// Per-region instance cap from the workspace limit, or the free-tier default
// until limits load.
const useReplicasMax = () => {
  const { limits } = useWorkspace();
  return Math.max(
    REPLICAS_MIN,
    limits?.autoscalingReplicasMax ?? freeTierLimits.autoscalingReplicasMax,
  );
};

const formatRangeParts = (replicasMin: number, replicasMax: number) => ({
  value: replicasMin === replicasMax ? String(replicasMax) : `${replicasMin} – ${replicasMax}`,
  unit: "",
});

const makeRangeSchema = (limit: number) =>
  z
    .object({
      replicasMin: z.number().int().min(REPLICAS_MIN).max(limit),
      replicasMax: z.number().int().min(REPLICAS_MIN).max(limit),
    })
    .refine((d) => d.replicasMin <= d.replicasMax, {
      message: "replicasMin must be ≤ replicasMax",
      path: ["replicasMin"],
    });

type RangeFormValues = z.infer<ReturnType<typeof makeRangeSchema>>;

const buildSliderRangeStyle = (replicasMin: number, replicasMax: number, limit: number) => {
  const span = limit - REPLICAS_MIN;
  const left = span > 0 ? (replicasMin - REPLICAS_MIN) / span : 0;
  const right = span > 0 ? (replicasMax - REPLICAS_MIN) / span : 0;
  return {
    background: `linear-gradient(to right, ${rampColorVar(`${COLOR_VAR}-4`)}, ${rampColorVar(`${COLOR_VAR}-12`)})`,
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

const noRegionsCheck = (settings: EnvironmentSettings): SaveState | null =>
  settings.regions.length > 0
    ? null
    : { status: "disabled", reason: "Select at least one region before setting instance count" };

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
  const { settings, variant } = useEnvironmentSettings();
  const updateEnvironment = useUpdateEnvironment();
  const replicasMaxLimit = useReplicasMax();
  const defaultValues = readRange(settings);

  const rangeSchema = useMemo(() => makeRangeSchema(replicasMaxLimit), [replicasMaxLimit]);

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
    updateEnvironment((draft) => {
      writeRange(draft, values);
    });
  };

  const hasChanges =
    currentReplicasMin !== defaultValues.replicasMin ||
    currentReplicasMax !== defaultValues.replicasMax;
  const extraCheck = noRegionsCheck(settings);
  const saveState = resolveSaveState([
    ...(extraCheck ? [[true, extraCheck] as [boolean, SaveState]] : []),
    [isSubmitting, { status: "saving" }],
    [!isValid, { status: "disabled" }],
    [!hasChanges, { status: "disabled", reason: "No changes to save" }],
  ]);

  return (
    <FormSettingCard
      title="Instances"
      description="Instances per region, scaled on CPU."
      onSubmit={handleSubmit(onSubmit)}
      saveState={saveState}
      autoSave={variant === "onboarding"}
    >
      <WideContent>
        <div className="flex items-center gap-3">
          <Slider
            min={REPLICAS_MIN}
            max={replicasMaxLimit}
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
            onValueCommitted={
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
                    updateEnvironment((draft) => {
                      writeRange(draft, { replicasMin, replicasMax });
                    });
                  }
                : undefined
            }
            className="flex-1 max-w-(--setting-w)"
            rangeStyle={buildSliderRangeStyle(
              currentReplicasMin,
              currentReplicasMax,
              replicasMaxLimit,
            )}
          />
          <RegionFlags settings={settings} />
          <span className="text-sm font-medium text-gray-12">
            {formatRangeParts(currentReplicasMin, currentReplicasMax).value}
          </span>
          <span className="text-xs text-gray-11">Up to {replicasMaxLimit}</span>
        </div>
      </WideContent>
    </FormSettingCard>
  );
};

"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { Gauge } from "@unkey/icons";
import { Slider } from "@unkey/ui";
import { useForm, useWatch } from "react-hook-form";
import { z } from "zod";
import { EditableSettingCard } from "../shared/editable-setting-card";
import { SettingDescription } from "../shared/setting-description";

const scalingSchema = z
  .object({
    minInstances: z.number().min(1).max(20),
    maxInstances: z.number().min(1).max(20),
    cpuThreshold: z.number().min(10).max(100),
  })
  .refine((d) => d.maxInstances >= d.minInstances, {
    message: "Max must be ≥ min",
    path: ["maxInstances"],
  });

type ScalingFormValues = z.infer<typeof scalingSchema>;

const DEFAULT_VALUES: ScalingFormValues = {
  minInstances: 1,
  maxInstances: 5,
  cpuThreshold: 80,
};

export const Scaling = () => {
  const {
    handleSubmit,
    setValue,
    formState: { isValid },
    control,
  } = useForm<ScalingFormValues>({
    resolver: zodResolver(scalingSchema),
    mode: "onChange",
    defaultValues: DEFAULT_VALUES,
  });

  const currentMin = useWatch({ control, name: "minInstances" });
  const currentMax = useWatch({ control, name: "maxInstances" });
  const currentCpuThreshold = useWatch({ control, name: "cpuThreshold" });

  const hasChanges =
    currentMin !== DEFAULT_VALUES.minInstances ||
    currentMax !== DEFAULT_VALUES.maxInstances ||
    currentCpuThreshold !== DEFAULT_VALUES.cpuThreshold;

  const onSubmit = (_values: ScalingFormValues) => {
    // no-op: backend not wired yet
  };

  return (
    <EditableSettingCard
      icon={<Gauge className="text-gray-12" iconSize="xl-medium" />}
      title="Scaling"
      border="bottom"
      description="Autoscaling instance range and CPU trigger threshold"
      displayValue={
        <div className="space-x-1">
          <span className="font-medium text-gray-12">
            {DEFAULT_VALUES.minInstances} – {DEFAULT_VALUES.maxInstances}
          </span>
          <span className="text-gray-11 font-normal">instances</span>
          <span className="text-gray-11 font-normal">·</span>
          <span className="font-medium text-gray-12">{DEFAULT_VALUES.cpuThreshold}%</span>
          <span className="text-gray-11 font-normal">CPU</span>
        </div>
      }
      onSubmit={(e) => e.preventDefault()}
      canSave={isValid && hasChanges}
      isSaving={false}
    >
      <div className="flex flex-col gap-4">
          <div className="flex flex-col">
            <span className="text-gray-11 text-[13px]">Autoscale range</span>
            <div className="flex items-center gap-3">
              <Slider
                min={1}
                max={20}
                step={1}
                value={[currentMin, currentMax]}
                onValueChange={([min, max]) => {
                  if (min !== undefined) {
                    setValue("minInstances", min, { shouldValidate: true });
                  }
                  if (max !== undefined) {
                    setValue("maxInstances", max, { shouldValidate: true });
                  }
                }}
                className="flex-1 max-w-[480px]"
                rangeStyle={{
                  background:
                    "linear-gradient(to right, hsla(var(--featureA-4)), hsla(var(--featureA-12)))",
                  backgroundRepeat: "no-repeat",
                }}
              />
              <span className="text-[13px]">
                <span className="font-medium text-gray-12">
                  {currentMin} – {currentMax}
                </span>{" "}
                <span className="text-gray-11 font-normal">instances</span>
              </span>
            </div>
            <SettingDescription>
              Minimum and maximum number of instances across all regions. Autoscaler stays within
              this range.
            </SettingDescription>
          </div>
          <div className="flex flex-col">
            <span className="text-gray-11 text-[13px]">CPU threshold</span>
            <div className="flex items-center gap-3">
              <Slider
                min={10}
                max={100}
                step={5}
                value={[currentCpuThreshold]}
                onValueChange={([value]) => {
                  if (value !== undefined) {
                    setValue("cpuThreshold", value, { shouldValidate: true });
                  }
                }}
                className="flex-1 max-w-[480px]"
                rangeStyle={{
                  background:
                    "linear-gradient(to right, hsla(var(--warningA-4)), hsla(var(--warningA-12)))",
                  backgroundRepeat: "no-repeat",
                }}
              />
              <span className="text-[13px]">
                <span className="font-medium text-gray-12">{currentCpuThreshold}%</span>
              </span>
            </div>
            <SettingDescription>
              Scale up when average CPU across instances exceeds this percentage. Changes apply on
              next deploy.
            </SettingDescription>
          </div>
        </div>
    </EditableSettingCard>
  );
};

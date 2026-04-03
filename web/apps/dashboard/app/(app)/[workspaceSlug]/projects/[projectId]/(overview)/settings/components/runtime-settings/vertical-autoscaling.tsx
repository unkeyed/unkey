"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { Gauge } from "@unkey/icons";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue, Slider } from "@unkey/ui";
import { Controller, useForm, useWatch } from "react-hook-form";
import { z } from "zod";
import { useEnvironmentSettings } from "../../environment-provider";
import { useUpdateAllEnvironments } from "../../hooks/use-update-all-environments";
import { SettingDescription, SettingField } from "../shared/form-blocks";
import { FormSettingCard, resolveSaveState } from "../shared/form-setting-card";

const updateModes = ["off", "initial", "recreate", "in_place_or_recreate"] as const;
const controlledResourceOptions = ["cpu", "memory", "both"] as const;
const controlledValueOptions = ["requests", "requests_and_limits"] as const;

const vpaSchema = z.object({
  enabled: z.boolean(),
  updateMode: z.enum(updateModes),
  controlledResources: z.enum(controlledResourceOptions),
  controlledValues: z.enum(controlledValueOptions),
  cpuMinMillicores: z.number().int().positive().nullable(),
  cpuMaxMillicores: z.number().int().positive().nullable(),
  memoryMinMib: z.number().int().positive().nullable(),
  memoryMaxMib: z.number().int().positive().nullable(),
});

type VpaFormValues = z.infer<typeof vpaSchema>;

const MODE_LABELS: Record<(typeof updateModes)[number], string> = {
  off: "Off (recommendations only)",
  initial: "Initial (set on pod creation)",
  recreate: "Recreate (evict and resize)",
  in_place_or_recreate: "In-Place or Recreate",
};

const RESOURCE_LABELS: Record<(typeof controlledResourceOptions)[number], string> = {
  cpu: "CPU only",
  memory: "Memory only",
  both: "CPU and Memory",
};

const VALUE_LABELS: Record<(typeof controlledValueOptions)[number], string> = {
  requests: "Requests only",
  requests_and_limits: "Requests and Limits",
};

function formatDisplayValue(values: VpaFormValues): string {
  if (!values.enabled) {
    return "Off";
  }
  const mode = MODE_LABELS[values.updateMode].split(" (")[0];
  const resources = RESOURCE_LABELS[values.controlledResources];
  return `${mode} · ${resources}`;
}

export const VerticalAutoscaling = () => {
  const { settings, variant } = useEnvironmentSettings();
  const defaults = settings.verticalAutoscaling;
  const updateAllEnvironments = useUpdateAllEnvironments();

  const {
    handleSubmit,
    control,
    setValue,
    formState: { isValid, isSubmitting },
  } = useForm<VpaFormValues>({
    resolver: zodResolver(vpaSchema),
    mode: "onChange",
    defaultValues: defaults,
  });

  const current = useWatch({ control });

  const hasChanges = JSON.stringify(current) !== JSON.stringify(defaults);

  const saveState = resolveSaveState([
    [isSubmitting, { status: "saving" }],
    [!isValid, { status: "disabled" }],
    [!hasChanges, { status: "disabled", reason: "No changes to save" }],
  ]);

  const onSubmit = (values: VpaFormValues) => {
    updateAllEnvironments((draft) => {
      draft.verticalAutoscaling = values;
    });
  };

  const enabled = current.enabled ?? defaults.enabled;

  return (
    <FormSettingCard
      icon={<Gauge className="text-gray-12" iconSize="xl-medium" />}
      title="Vertical Autoscaling"
      description="Automatically right-size CPU and memory based on actual usage"
      displayValue={
        <span className="font-medium text-gray-12">
          {formatDisplayValue(current as VpaFormValues)}
        </span>
      }
      onSubmit={handleSubmit(onSubmit)}
      saveState={saveState}
      autoSave={variant === "onboarding"}
    >
      <SettingField>
        <div className="flex items-center gap-3">
          <span className="text-gray-11 text-[13px]">Enable VPA</span>
          <button
            type="button"
            role="switch"
            aria-checked={enabled}
            onClick={() => setValue("enabled", !enabled, { shouldValidate: true })}
            className={`relative inline-flex h-5 w-9 shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors ${
              enabled ? "bg-accent-9" : "bg-gray-6"
            }`}
          >
            <span
              className={`pointer-events-none inline-block size-4 rounded-full bg-white shadow-sm transition-transform ${
                enabled ? "translate-x-4" : "translate-x-0"
              }`}
            />
          </button>
        </div>
        <SettingDescription>
          When enabled, VPA adjusts resource requests based on observed usage. A workload uses
          either horizontal (HPA) or vertical (VPA) autoscaling, not both.
        </SettingDescription>
      </SettingField>

      {enabled && (
        <>
          <SettingField>
            <span className="text-gray-11 text-[13px]">Update mode</span>
            <Controller
              name="updateMode"
              control={control}
              render={({ field }) => (
                <Select value={field.value} onValueChange={field.onChange}>
                  <SelectTrigger className="max-w-[var(--setting-w)]">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {updateModes.map((mode) => (
                      <SelectItem key={mode} value={mode}>
                        {MODE_LABELS[mode]}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              )}
            />
            <SettingDescription>
              Off: recommendations only. Initial: applied at pod creation. Recreate: evicts pods to
              resize. In-Place or Recreate: resizes without eviction when possible.
            </SettingDescription>
          </SettingField>

          <SettingField>
            <span className="text-gray-11 text-[13px]">Controlled resources</span>
            <Controller
              name="controlledResources"
              control={control}
              render={({ field }) => (
                <Select value={field.value} onValueChange={field.onChange}>
                  <SelectTrigger className="max-w-[var(--setting-w)]">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {controlledResourceOptions.map((opt) => (
                      <SelectItem key={opt} value={opt}>
                        {RESOURCE_LABELS[opt]}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              )}
            />
            <SettingDescription>Which resources VPA is allowed to adjust.</SettingDescription>
          </SettingField>

          <SettingField>
            <span className="text-gray-11 text-[13px]">Controlled values</span>
            <Controller
              name="controlledValues"
              control={control}
              render={({ field }) => (
                <Select value={field.value} onValueChange={field.onChange}>
                  <SelectTrigger className="max-w-[var(--setting-w)]">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {controlledValueOptions.map((opt) => (
                      <SelectItem key={opt} value={opt}>
                        {VALUE_LABELS[opt]}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              )}
            />
            <SettingDescription>
              "Requests only" is safer — VPA adjusts scheduling requests but leaves limits
              unchanged.
            </SettingDescription>
          </SettingField>

          <SettingField>
            <span className="text-gray-11 text-[13px]">CPU bounds (millicores)</span>
            <div className="flex items-center gap-3">
              <Slider
                min={50}
                max={4000}
                step={50}
                value={[current.cpuMinMillicores ?? 50, current.cpuMaxMillicores ?? 4000]}
                onValueChange={([min, max]) => {
                  if (min !== undefined) {
                    setValue("cpuMinMillicores", min, { shouldValidate: true });
                  }
                  if (max !== undefined) {
                    setValue("cpuMaxMillicores", max, { shouldValidate: true });
                  }
                }}
                className="flex-1 max-w-[var(--setting-w)]"
                rangeStyle={{
                  background:
                    "linear-gradient(to right, hsla(var(--infoA-4)), hsla(var(--infoA-12)))",
                  backgroundRepeat: "no-repeat",
                }}
              />
              <span className="text-[13px] whitespace-nowrap">
                <span className="font-medium text-gray-12">
                  {current.cpuMinMillicores ?? 50}m – {current.cpuMaxMillicores ?? 4000}m
                </span>
              </span>
            </div>
            <SettingDescription>
              VPA will never recommend CPU outside these bounds.
            </SettingDescription>
          </SettingField>

          <SettingField>
            <span className="text-gray-11 text-[13px]">Memory bounds (MiB)</span>
            <div className="flex items-center gap-3">
              <Slider
                min={64}
                max={4096}
                step={64}
                value={[current.memoryMinMib ?? 64, current.memoryMaxMib ?? 4096]}
                onValueChange={([min, max]) => {
                  if (min !== undefined) {
                    setValue("memoryMinMib", min, { shouldValidate: true });
                  }
                  if (max !== undefined) {
                    setValue("memoryMaxMib", max, { shouldValidate: true });
                  }
                }}
                className="flex-1 max-w-[var(--setting-w)]"
                rangeStyle={{
                  background:
                    "linear-gradient(to right, hsla(var(--warningA-4)), hsla(var(--warningA-12)))",
                  backgroundRepeat: "no-repeat",
                }}
              />
              <span className="text-[13px] whitespace-nowrap">
                <span className="font-medium text-gray-12">
                  {current.memoryMinMib ?? 64} – {current.memoryMaxMib ?? 4096} MiB
                </span>
              </span>
            </div>
            <SettingDescription>
              VPA will never recommend memory outside these bounds.
            </SettingDescription>
          </SettingField>
        </>
      )}
    </FormSettingCard>
  );
};

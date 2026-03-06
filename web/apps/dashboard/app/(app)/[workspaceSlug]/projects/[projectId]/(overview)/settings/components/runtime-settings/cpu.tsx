"use client";

import { collection } from "@/lib/collections";
import { formatCpuParts } from "@/lib/utils/deployment-formatters";
import { zodResolver } from "@hookform/resolvers/zod";
import { Bolt } from "@unkey/icons";
import { Slider } from "@unkey/ui";
import { useContext, useEffect } from "react";
import { useForm, useWatch } from "react-hook-form";
import { z } from "zod";
import { EnvironmentContext, useEnvironmentSettings } from "../../environment-provider";
import { useMultiEnvironmentSettings } from "../../hooks/use-multi-environment-settings";
import { EnvironmentSliderSection } from "../shared/environment-slider-section";
import { FormSettingCard, resolveSaveState } from "../shared/form-setting-card";
import { SettingDescription } from "../shared/setting-description";
import { indexToValue, valueToIndex } from "../shared/slider-utils";

const CPU_OPTIONS = [
  { label: "1/4 vCPU", value: 256 },
  { label: "1/2 vCPU", value: 512 },
  { label: "1 vCPU", value: 1024 },
  { label: "2 vCPU", value: 2048 },
  { label: "4 vCPU", value: 4096 },
  { label: "8 vCPU", value: 8192 },
  { label: "16 vCPU", value: 16384 },
  { label: "32 vCPU", value: 32768 },
] as const;

export const Cpu = () => {
  const envContext = useContext(EnvironmentContext);

  if (envContext?.variant === "onboarding") {
    return <CpuSingle />;
  }

  return <CpuDual />;
};

const cpuSingleSchema = z.object({ cpu: z.number() });
type CpuSingleFormValues = z.infer<typeof cpuSingleSchema>;

const CpuSingle = () => {
  const { settings, variant } = useEnvironmentSettings();
  const defaultCpu = settings.cpuMillicores;

  const {
    handleSubmit,
    setValue,
    formState: { isValid, isSubmitting },
    control,
    reset,
  } = useForm<CpuSingleFormValues>({
    resolver: zodResolver(cpuSingleSchema),
    mode: "onChange",
    defaultValues: { cpu: defaultCpu },
  });

  useEffect(() => {
    reset({ cpu: defaultCpu });
  }, [defaultCpu, reset]);

  const currentCpu = useWatch({ control, name: "cpu" });

  const onSubmit = async (values: CpuSingleFormValues) => {
    collection.environmentSettings.update(settings.environmentId, (draft) => {
      draft.cpuMillicores = values.cpu;
    });
  };

  const hasChanges = currentCpu !== defaultCpu;
  const cpuIndex = valueToIndex(CPU_OPTIONS, currentCpu);

  const saveState = resolveSaveState([
    [isSubmitting, { status: "saving" }],
    [!isValid, { status: "disabled" }],
    [!hasChanges, { status: "disabled", reason: "No changes to save" }],
  ]);

  return (
    <FormSettingCard
      icon={<Bolt className="text-gray-12" iconSize="xl-medium" />}
      title="CPU"
      description="CPU allocation for each instance"
      displayValue={
        <span>
          <span className="font-medium text-gray-12">{formatCpuParts(defaultCpu).value}</span>{" "}
          <span className="text-gray-11">{formatCpuParts(defaultCpu).unit}</span>
        </span>
      }
      onSubmit={handleSubmit(onSubmit)}
      saveState={saveState}
      autoSave
    >
      <div className="flex items-center gap-3">
        <Slider
          min={0}
          max={CPU_OPTIONS.length - 1}
          step={1}
          value={[cpuIndex]}
          onValueChange={([value]) => {
            if (value !== undefined) {
              setValue("cpu", indexToValue(CPU_OPTIONS, value, 256), {
                shouldValidate: true,
              });
            }
          }}
          onValueCommit={
            variant === "onboarding"
              ? ([value]) => {
                  if (value !== undefined) {
                    const cpu = indexToValue(CPU_OPTIONS, value, 256);
                    if (cpu !== defaultCpu) {
                      collection.environmentSettings.update(settings.environmentId, (draft) => {
                        draft.cpuMillicores = cpu;
                      });
                    }
                  }
                }
              : undefined
          }
          className="flex-1 max-w-[480px]"
          rangeStyle={{
            background: "linear-gradient(to right, hsla(var(--infoA-4)), hsla(var(--infoA-12)))",
            backgroundSize: `${cpuIndex > 0 ? 100 / (cpuIndex / (CPU_OPTIONS.length - 1)) : 100}% 100%`,
            backgroundRepeat: "no-repeat",
          }}
        />
        <span className="text-[13px]">
          <span className="font-medium text-gray-12">{formatCpuParts(currentCpu).value}</span>{" "}
          <span className="text-gray-11">{formatCpuParts(currentCpu).unit}</span>
        </span>
      </div>

      <SettingDescription>
        Higher CPU improves compute-heavy workloads. Changes apply on next deploy.
      </SettingDescription>
    </FormSettingCard>
  );
};

const cpuDualSchema = z.object({
  productionCpu: z.number(),
  previewCpu: z.number(),
});

type CpuDualFormValues = z.infer<typeof cpuDualSchema>;

const CpuDual = () => {
  const multiSettings = useMultiEnvironmentSettings();

  if (!multiSettings) {
    return null;
  }

  return <CpuDualInner production={multiSettings.production} preview={multiSettings.preview} />;
};

type CpuDualInnerProps = {
  production: { environmentId: string; cpuMillicores: number };
  preview: { environmentId: string; cpuMillicores: number };
};

const CpuDualInner = ({ production, preview }: CpuDualInnerProps) => {
  const defaultProdCpu = production.cpuMillicores;
  const defaultPreviewCpu = preview.cpuMillicores;

  const {
    handleSubmit,
    setValue,
    formState: { isValid, isSubmitting },
    control,
    reset,
  } = useForm<CpuDualFormValues>({
    resolver: zodResolver(cpuDualSchema),
    mode: "onChange",
    defaultValues: { productionCpu: defaultProdCpu, previewCpu: defaultPreviewCpu },
  });

  useEffect(() => {
    reset({ productionCpu: defaultProdCpu, previewCpu: defaultPreviewCpu });
  }, [defaultProdCpu, defaultPreviewCpu, reset]);

  const currentProdCpu = useWatch({ control, name: "productionCpu" });
  const currentPreviewCpu = useWatch({ control, name: "previewCpu" });

  const onSubmit = async (values: CpuDualFormValues) => {
    if (values.productionCpu !== defaultProdCpu) {
      collection.environmentSettings.update(production.environmentId, (draft) => {
        draft.cpuMillicores = values.productionCpu;
      });
    }
    if (values.previewCpu !== defaultPreviewCpu) {
      collection.environmentSettings.update(preview.environmentId, (draft) => {
        draft.cpuMillicores = values.previewCpu;
      });
    }
  };

  const prodChanged = currentProdCpu !== defaultProdCpu;
  const previewChanged = currentPreviewCpu !== defaultPreviewCpu;
  const hasChanges = prodChanged || previewChanged;

  const prodIndex = valueToIndex(CPU_OPTIONS, currentProdCpu);
  const previewIndex = valueToIndex(CPU_OPTIONS, currentPreviewCpu);

  const saveState = resolveSaveState([
    [isSubmitting, { status: "saving" }],
    [!isValid, { status: "disabled" }],
    [!hasChanges, { status: "disabled", reason: "No changes to save" }],
  ]);

  return (
    <FormSettingCard
      icon={<Bolt className="text-gray-12" iconSize="xl-medium" />}
      title="CPU"
      description="CPU allocation for each instance"
      displayValue={
        <div className="flex items-center gap-3">
          <EnvironmentDisplayValue label="Production" cpu={defaultProdCpu} />
          <span className="text-gray-8">|</span>
          <EnvironmentDisplayValue label="Preview" cpu={defaultPreviewCpu} />
        </div>
      }
      onSubmit={handleSubmit(onSubmit)}
      saveState={saveState}
    >
      <EnvironmentSliderSection label="Production">
        <div className="flex items-center gap-3">
          <Slider
            min={0}
            max={CPU_OPTIONS.length - 1}
            step={1}
            value={[prodIndex]}
            onValueChange={([value]) => {
              if (value !== undefined) {
                setValue("productionCpu", indexToValue(CPU_OPTIONS, value, 256), {
                  shouldValidate: true,
                });
              }
            }}
            className="flex-1 max-w-[480px]"
            rangeStyle={{
              background: "linear-gradient(to right, hsla(var(--infoA-4)), hsla(var(--infoA-12)))",
              backgroundSize: `${prodIndex > 0 ? 100 / (prodIndex / (CPU_OPTIONS.length - 1)) : 100}% 100%`,
              backgroundRepeat: "no-repeat",
            }}
          />
          <span className="text-[13px]">
            <span className="font-medium text-gray-12">{formatCpuParts(currentProdCpu).value}</span>{" "}
            <span className="text-gray-11">{formatCpuParts(currentProdCpu).unit}</span>
          </span>
        </div>
      </EnvironmentSliderSection>

      <EnvironmentSliderSection label="Preview">
        <div className="flex items-center gap-3">
          <Slider
            min={0}
            max={CPU_OPTIONS.length - 1}
            step={1}
            value={[previewIndex]}
            onValueChange={([value]) => {
              if (value !== undefined) {
                setValue("previewCpu", indexToValue(CPU_OPTIONS, value, 256), {
                  shouldValidate: true,
                });
              }
            }}
            className="flex-1 max-w-[480px]"
            rangeStyle={{
              background: "linear-gradient(to right, hsla(var(--infoA-4)), hsla(var(--infoA-12)))",
              backgroundSize: `${previewIndex > 0 ? 100 / (previewIndex / (CPU_OPTIONS.length - 1)) : 100}% 100%`,
              backgroundRepeat: "no-repeat",
            }}
          />
          <span className="text-[13px]">
            <span className="font-medium text-gray-12">
              {formatCpuParts(currentPreviewCpu).value}
            </span>{" "}
            <span className="text-gray-11">{formatCpuParts(currentPreviewCpu).unit}</span>
          </span>
        </div>
      </EnvironmentSliderSection>

      <SettingDescription>
        Higher CPU improves compute-heavy workloads. Changes apply on next deploy.
      </SettingDescription>
    </FormSettingCard>
  );
};

const EnvironmentDisplayValue = ({ label, cpu }: { label: string; cpu: number }) => {
  const parts = formatCpuParts(cpu);
  return (
    <div className="space-x-1">
      <span className="text-gray-11 text-xs font-normal">{label}</span>
      <span className="font-medium text-gray-12">{parts.value}</span>
      <span className="text-gray-11 font-normal">{parts.unit}</span>
    </div>
  );
};

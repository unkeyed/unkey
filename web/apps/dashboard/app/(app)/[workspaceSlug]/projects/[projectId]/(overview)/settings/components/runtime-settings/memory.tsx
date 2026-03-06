"use client";

import { collection } from "@/lib/collections";
import { formatMemoryParts } from "@/lib/utils/deployment-formatters";
import { zodResolver } from "@hookform/resolvers/zod";
import { ScanCode } from "@unkey/icons";
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

const MEMORY_OPTIONS = [
  { label: "256 MiB", value: 256 },
  { label: "512 MiB", value: 512 },
  { label: "1 GiB", value: 1024 },
  { label: "2 GiB", value: 2048 },
  { label: "4 GiB", value: 4096 },
  { label: "8 GiB", value: 8192 },
  { label: "16 GiB", value: 16384 },
  { label: "32 GiB", value: 32768 },
] as const;

export const Memory = () => {
  const envContext = useContext(EnvironmentContext);

  if (envContext?.variant === "onboarding") {
    return <MemorySingle />;
  }

  return <MemoryDual />;
};

const memorySingleSchema = z.object({ memory: z.number() });
type MemorySingleFormValues = z.infer<typeof memorySingleSchema>;

const MemorySingle = () => {
  const { settings, variant } = useEnvironmentSettings();
  const defaultMemory = settings.memoryMib;

  const {
    handleSubmit,
    setValue,
    formState: { isValid, isSubmitting },
    control,
    reset,
  } = useForm<MemorySingleFormValues>({
    resolver: zodResolver(memorySingleSchema),
    mode: "onChange",
    defaultValues: { memory: defaultMemory },
  });

  useEffect(() => {
    reset({ memory: defaultMemory });
  }, [defaultMemory, reset]);

  const currentMemory = useWatch({ control, name: "memory" });

  const onSubmit = async (values: MemorySingleFormValues) => {
    collection.environmentSettings.update(settings.environmentId, (draft) => {
      draft.memoryMib = values.memory;
    });
  };

  const hasChanges = currentMemory !== defaultMemory;
  const memoryIndex = valueToIndex(MEMORY_OPTIONS, currentMemory);

  const saveState = resolveSaveState([
    [isSubmitting, { status: "saving" }],
    [!isValid, { status: "disabled" }],
    [!hasChanges, { status: "disabled", reason: "No changes to save" }],
  ]);

  return (
    <FormSettingCard
      icon={<ScanCode className="text-gray-12" iconSize="xl-medium" />}
      title="Memory"
      description="Memory allocation for each instance"
      displayValue={
        <span>
          <span className="font-medium text-gray-12">{formatMemoryParts(defaultMemory).value}</span>{" "}
          <span className="text-gray-11">{formatMemoryParts(defaultMemory).unit}</span>
        </span>
      }
      onSubmit={handleSubmit(onSubmit)}
      saveState={saveState}
      autoSave
    >
      <div className="flex items-center gap-3">
        <Slider
          min={0}
          max={MEMORY_OPTIONS.length - 1}
          step={1}
          value={[memoryIndex]}
          onValueChange={([value]) => {
            if (value !== undefined) {
              setValue("memory", indexToValue(MEMORY_OPTIONS, value, 256), {
                shouldValidate: true,
              });
            }
          }}
          onValueCommit={
            variant === "onboarding"
              ? ([value]) => {
                if (value !== undefined) {
                  const memory = indexToValue(MEMORY_OPTIONS, value, 256);
                  if (memory !== defaultMemory) {
                    collection.environmentSettings.update(settings.environmentId, (draft) => {
                      draft.memoryMib = memory;
                    });
                  }
                }
              }
              : undefined
          }
          className="flex-1 max-w-[480px]"
          rangeStyle={{
            background:
              "linear-gradient(to right, hsla(var(--warningA-4)), hsla(var(--warningA-12)))",
            backgroundSize: `${memoryIndex > 0 ? 100 / (memoryIndex / (MEMORY_OPTIONS.length - 1)) : 100}% 100%`,
            backgroundRepeat: "no-repeat",
          }}
        />
        <span className="text-[13px]">
          <span className="font-medium text-gray-12">{formatMemoryParts(currentMemory).value}</span>{" "}
          <span className="text-gray-11">{formatMemoryParts(currentMemory).unit}</span>
        </span>
      </div>

      <SettingDescription>
        Increase memory for applications with large datasets or caching needs. Changes apply on next
        deploy.
      </SettingDescription>
    </FormSettingCard>
  );
};

const memoryDualSchema = z.object({
  productionMemory: z.number(),
  previewMemory: z.number(),
});

type MemoryDualFormValues = z.infer<typeof memoryDualSchema>;

const MemoryDual = () => {
  const multiSettings = useMultiEnvironmentSettings();

  if (!multiSettings) {
    return null;
  }

  return <MemoryDualInner production={multiSettings.production} preview={multiSettings.preview} />;
};

type MemoryDualInnerProps = {
  production: { environmentId: string; memoryMib: number };
  preview: { environmentId: string; memoryMib: number };
};

const MemoryDualInner = ({ production, preview }: MemoryDualInnerProps) => {
  const defaultProdMemory = production.memoryMib;
  const defaultPreviewMemory = preview.memoryMib;

  const {
    handleSubmit,
    setValue,
    formState: { isValid, isSubmitting },
    control,
    reset,
  } = useForm<MemoryDualFormValues>({
    resolver: zodResolver(memoryDualSchema),
    mode: "onChange",
    defaultValues: { productionMemory: defaultProdMemory, previewMemory: defaultPreviewMemory },
  });

  useEffect(() => {
    reset({ productionMemory: defaultProdMemory, previewMemory: defaultPreviewMemory });
  }, [defaultProdMemory, defaultPreviewMemory, reset]);

  const currentProdMemory = useWatch({ control, name: "productionMemory" });
  const currentPreviewMemory = useWatch({ control, name: "previewMemory" });

  const onSubmit = async (values: MemoryDualFormValues) => {
    if (values.productionMemory !== defaultProdMemory) {
      collection.environmentSettings.update(production.environmentId, (draft) => {
        draft.memoryMib = values.productionMemory;
      });
    }
    if (values.previewMemory !== defaultPreviewMemory) {
      collection.environmentSettings.update(preview.environmentId, (draft) => {
        draft.memoryMib = values.previewMemory;
      });
    }
  };

  const prodChanged = currentProdMemory !== defaultProdMemory;
  const previewChanged = currentPreviewMemory !== defaultPreviewMemory;
  const hasChanges = prodChanged || previewChanged;

  const prodIndex = valueToIndex(MEMORY_OPTIONS, currentProdMemory);
  const previewIndex = valueToIndex(MEMORY_OPTIONS, currentPreviewMemory);

  const saveState = resolveSaveState([
    [isSubmitting, { status: "saving" }],
    [!isValid, { status: "disabled" }],
    [!hasChanges, { status: "disabled", reason: "No changes to save" }],
  ]);

  return (
    <FormSettingCard
      icon={<ScanCode className="text-gray-12" iconSize="xl-medium" />}
      title="Memory"
      description="Memory allocation for each instance"
      displayValue={
        <div className="flex items-center gap-3">
          <EnvironmentDisplayValue label="Production" memory={defaultProdMemory} />
          <span className="text-gray-8">|</span>
          <EnvironmentDisplayValue label="Preview" memory={defaultPreviewMemory} />
        </div>
      }
      onSubmit={handleSubmit(onSubmit)}
      saveState={saveState}
    >
      <EnvironmentSliderSection label="Production">
        <div className="flex items-center gap-3">
          <Slider
            min={0}
            max={MEMORY_OPTIONS.length - 1}
            step={1}
            value={[prodIndex]}
            onValueChange={([value]) => {
              if (value !== undefined) {
                setValue("productionMemory", indexToValue(MEMORY_OPTIONS, value, 256), {
                  shouldValidate: true,
                });
              }
            }}
            className="flex-1 max-w-[480px]"
            rangeStyle={{
              background:
                "linear-gradient(to right, hsla(var(--warningA-4)), hsla(var(--warningA-12)))",
              backgroundSize: `${prodIndex > 0 ? 100 / (prodIndex / (MEMORY_OPTIONS.length - 1)) : 100}% 100%`,
              backgroundRepeat: "no-repeat",
            }}
          />
          <span className="text-[13px]">
            <span className="font-medium text-gray-12">
              {formatMemoryParts(currentProdMemory).value}
            </span>{" "}
            <span className="text-gray-11">{formatMemoryParts(currentProdMemory).unit}</span>
          </span>
        </div>
      </EnvironmentSliderSection>

      <EnvironmentSliderSection label="Preview">
        <div className="flex items-center gap-3">
          <Slider
            min={0}
            max={MEMORY_OPTIONS.length - 1}
            step={1}
            value={[previewIndex]}
            onValueChange={([value]) => {
              if (value !== undefined) {
                setValue("previewMemory", indexToValue(MEMORY_OPTIONS, value, 256), {
                  shouldValidate: true,
                });
              }
            }}
            className="flex-1 max-w-[480px]"
            rangeStyle={{
              background:
                "linear-gradient(to right, hsla(var(--warningA-4)), hsla(var(--warningA-12)))",
              backgroundSize: `${previewIndex > 0 ? 100 / (previewIndex / (MEMORY_OPTIONS.length - 1)) : 100}% 100%`,
              backgroundRepeat: "no-repeat",
            }}
          />
          <span className="text-[13px]">
            <span className="font-medium text-gray-12">
              {formatMemoryParts(currentPreviewMemory).value}
            </span>{" "}
            <span className="text-gray-11">{formatMemoryParts(currentPreviewMemory).unit}</span>
          </span>
        </div>
      </EnvironmentSliderSection>

      <SettingDescription>
        Increase memory for applications with large datasets or caching needs. Changes apply on next
        deploy.
      </SettingDescription>
    </FormSettingCard>
  );
};

const EnvironmentDisplayValue = ({ label, memory }: { label: string; memory: number }) => {
  const parts = formatMemoryParts(memory);
  return (
    <div className="space-x-1">
      <span className="text-gray-11 text-xs font-normal">{label}</span>
      <span className="font-medium text-gray-12">{parts.value}</span>
      <span className="text-gray-11 font-normal">{parts.unit}</span>
    </div>
  );
};

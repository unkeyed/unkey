"use client";

import { collection } from "@/lib/collections";
import { formatMemory } from "@/lib/utils/deployment-formatters";
import { zodResolver } from "@hookform/resolvers/zod";
import { eq, useLiveQuery } from "@tanstack/react-db";
import { ScanCode } from "@unkey/icons";
import { Slider } from "@unkey/ui";
import { useEffect } from "react";
import { useForm, useWatch } from "react-hook-form";
import { z } from "zod";
import { useEnvironmentId } from "../../environment-provider";
import { FormSettingCard } from "../shared/form-setting-card";
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

const memorySchema = z.object({
  memory: z.number(),
});

type MemoryFormValues = z.infer<typeof memorySchema>;

export const Memory = () => {
  const environmentId = useEnvironmentId();

  const { data: settings } = useLiveQuery(
    (q) =>
      q
        .from({ s: collection.environmentSettings })
        .where(({ s }) => eq(s.environmentId, environmentId)),
    [environmentId],
  );

  const defaultMemory = settings?.[0]?.memoryMib ?? 256;

  return <MemoryForm environmentId={environmentId} defaultMemory={defaultMemory} />;
};

type MemoryFormProps = {
  environmentId: string;
  defaultMemory: number;
};

const MemoryForm: React.FC<MemoryFormProps> = ({ environmentId, defaultMemory }) => {
  const {
    handleSubmit,
    setValue,
    formState: { isValid, isSubmitting },
    control,
    reset,
  } = useForm<MemoryFormValues>({
    resolver: zodResolver(memorySchema),
    mode: "onChange",
    defaultValues: { memory: defaultMemory },
  });

  useEffect(() => {
    reset({ memory: defaultMemory });
  }, [defaultMemory, reset]);

  const currentMemory = useWatch({ control, name: "memory" });

  const onSubmit = async (values: MemoryFormValues) => {
    collection.environmentSettings.update(environmentId, (draft) => {
      draft.memoryMib = values.memory;
    });
  };

  const hasChanges = currentMemory !== defaultMemory;
  const currentIndex = valueToIndex(MEMORY_OPTIONS, currentMemory);

  return (
    <FormSettingCard
      icon={<ScanCode className="text-gray-12" iconSize="xl-medium" />}
      title="Memory"
      description="Memory allocation for each instance"
      displayValue={(() => {
        const [value, unit] = parseMemoryDisplay(defaultMemory);
        return (
          <div className="space-x-1">
            <span className="font-medium text-gray-12">{value}</span>
            <span className="text-gray-11 font-normal">{unit}</span>
          </div>
        );
      })()}
      onSubmit={handleSubmit(onSubmit)}
      canSave={isValid && !isSubmitting && hasChanges}
      isSaving={isSubmitting}
    >
      <div className="flex flex-col">
        <span className="text-gray-11 text-[13px]">Memory per instance</span>
        <div className="flex items-center gap-3">
          <Slider
            min={0}
            max={MEMORY_OPTIONS.length - 1}
            step={1}
            value={[currentIndex]}
            onValueChange={([value]) => {
              if (value !== undefined) {
                setValue("memory", indexToValue(MEMORY_OPTIONS, value, 256), {
                  shouldValidate: true,
                });
              }
            }}
            className="flex-1 max-w-[480px]"
            rangeStyle={{
              background:
                "linear-gradient(to right, hsla(var(--warningA-4)), hsla(var(--warningA-12)))",
              backgroundSize: `${currentIndex > 0 ? 100 / (currentIndex / (MEMORY_OPTIONS.length - 1)) : 100}% 100%`,
              backgroundRepeat: "no-repeat",
            }}
          />
          <span className="text-[13px]">
            <span className="font-medium text-gray-12">{formatMemory(currentMemory)}</span>
          </span>
        </div>
        <SettingDescription>
          Increase memory for applications with large datasets or caching needs. Changes apply on
          next deploy.
        </SettingDescription>
      </div>
    </FormSettingCard>
  );
};

function parseMemoryDisplay(mib: number): [string, string] {
  if (mib >= 1024) {
    return [`${(mib / 1024).toFixed(mib % 1024 === 0 ? 0 : 1)}`, "GiB"];
  }
  return [`${mib}`, "MiB"];
}

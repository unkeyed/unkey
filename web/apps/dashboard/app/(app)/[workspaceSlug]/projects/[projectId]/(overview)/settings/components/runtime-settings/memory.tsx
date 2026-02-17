"use client";

import { formatMemory } from "@/lib/utils/deployment-formatters";
import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import { ScanCode } from "@unkey/icons";
import { Slider, toast } from "@unkey/ui";
import { useEffect } from "react";
import { useForm, useWatch } from "react-hook-form";
import { z } from "zod";
import { useProjectData } from "../../../data-provider";
import { EditableSettingCard } from "../shared/editable-setting-card";

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
  const { environments } = useProjectData();
  const environmentId = environments[0]?.id;

  const { data: settingsData } = trpc.deploy.environmentSettings.get.useQuery(
    { environmentId: environmentId ?? "" },
    { enabled: Boolean(environmentId) },
  );

  const defaultMemory = settingsData?.runtimeSettings?.memoryMib ?? 256;

  return <MemoryForm environmentId={environmentId} defaultMemory={defaultMemory} />;
};

type MemoryFormProps = {
  environmentId: string;
  defaultMemory: number;
};

const MemoryForm: React.FC<MemoryFormProps> = ({ environmentId, defaultMemory }) => {
  const utils = trpc.useUtils();

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

  const updateRuntime = trpc.deploy.environmentSettings.updateRuntime.useMutation({
    onSuccess: () => {
      utils.deploy.environmentSettings.get.invalidate({ environmentId });
    },
    onError: (err) => {
      toast.error("Failed to update memory", { description: err.message });
    },
  });

  const onSubmit = async (values: MemoryFormValues) => {
    await updateRuntime.mutateAsync({
      environmentId,
      memoryMib: values.memory,
    });
  };

  const hasChanges = currentMemory !== defaultMemory;
  const currentIndex = valueToIndex(currentMemory);

  return (
    <EditableSettingCard
      icon={<ScanCode className="text-gray-12" iconSize="xl-medium" />}
      title="Memory"
      description="Memory allocation for each instance"
      border="bottom"
      displayValue={(() => {
        const [value, unit] = parseMemoryDisplay(defaultMemory);
        return (
          <div className="space-x-1">
            <span className="font-medium text-gray-12">{value}</span>
            <span className="text-gray-11 font-normal">{unit}</span>
          </div>
        );
      })()}
      formId="update-memory-form"
      canSave={isValid && !isSubmitting && hasChanges}
      isSaving={updateRuntime.isLoading || isSubmitting}
    >
      <form id="update-memory-form" onSubmit={handleSubmit(onSubmit)}>
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
                  setValue("memory", indexToValue(value), { shouldValidate: true });
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
        </div>
      </form>
    </EditableSettingCard>
  );
};


function valueToIndex(mib: number): number {
  const idx = MEMORY_OPTIONS.findIndex((o) => o.value === mib);
  return idx >= 0 ? idx : 0;
}

function indexToValue(index: number): number {
  return MEMORY_OPTIONS[index]?.value ?? 256;
}

function parseMemoryDisplay(mib: number): [string, string] {
  if (mib >= 1024) {
    return [`${(mib / 1024).toFixed(mib % 1024 === 0 ? 0 : 1)}`, "GiB"];
  }
  return [`${mib}`, "MiB"];
}



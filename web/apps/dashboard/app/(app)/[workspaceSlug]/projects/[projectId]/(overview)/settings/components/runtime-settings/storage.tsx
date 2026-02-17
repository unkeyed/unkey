"use client";

import { formatMemory } from "@/lib/utils/deployment-formatters";
import { zodResolver } from "@hookform/resolvers/zod";
import { Harddrive } from "@unkey/icons";
import { Slider, toast } from "@unkey/ui";
import { useForm, useWatch } from "react-hook-form";
import { z } from "zod";
import { EditableSettingCard } from "../shared/editable-setting-card";
import { SettingDescription } from "../shared/setting-description";

const STORAGE_OPTIONS = [
  { label: "512 MiB", value: 512 },
  { label: "1 GiB", value: 1024 },
  { label: "2 GiB", value: 2048 },
  { label: "5 GiB", value: 5120 },
  { label: "10 GiB", value: 10240 },
  { label: "20 GiB", value: 20480 },
  { label: "50 GiB", value: 51200 },
] as const;

const DEFAULT_STORAGE_MIB = 1024;

const storageSchema = z.object({
  storage: z.number(),
});

type StorageFormValues = z.infer<typeof storageSchema>;

export const Storage = () => {
  return <StorageForm defaultStorage={DEFAULT_STORAGE_MIB} />;
};

type StorageFormProps = {
  defaultStorage: number;
};

const StorageForm: React.FC<StorageFormProps> = ({ defaultStorage }) => {
  const {
    handleSubmit,
    setValue,
    formState: { isValid },
    control,
  } = useForm<StorageFormValues>({
    resolver: zodResolver(storageSchema),
    mode: "onChange",
    defaultValues: { storage: defaultStorage },
  });

  const currentStorage = useWatch({ control, name: "storage" });

  const onSubmit = async (_values: StorageFormValues) => {
    // Backend storage field not yet implemented
    toast.info("Storage settings are not yet available");
  };

  const hasChanges = currentStorage !== defaultStorage;
  const currentIndex = valueToIndex(currentStorage);

  return (
    <EditableSettingCard
      icon={<Harddrive className="text-gray-12" iconSize="xl-medium" />}
      title="Storage"
      description="Ephemeral disk space per instance"
      displayValue={(() => {
        const [value, unit] = parseStorageDisplay(defaultStorage);
        return (
          <div className="space-x-1">
            <span className="font-medium text-gray-12">{value}</span>
            <span className="text-gray-11 font-normal">{unit}</span>
          </div>
        );
      })()}
      formId="update-storage-form"
      canSave={isValid && hasChanges}
      isSaving={false}
    >
      <form id="update-storage-form" onSubmit={handleSubmit(onSubmit)}>
        <div className="flex flex-col">
          <span className="text-gray-11 text-[13px]">Storage per instance</span>
          <div className="flex items-center gap-3">
            <Slider
              min={0}
              max={STORAGE_OPTIONS.length - 1}
              step={1}
              value={[currentIndex]}
              onValueChange={([value]) => {
                if (value !== undefined) {
                  setValue("storage", indexToValue(value), { shouldValidate: true });
                }
              }}
              className="flex-1 max-w-[480px]"
              rangeStyle={{
                background:
                  "linear-gradient(to right, hsla(var(--successA-4)), hsla(var(--successA-12)))",
                backgroundSize: `${currentIndex > 0 ? 100 / (currentIndex / (STORAGE_OPTIONS.length - 1)) : 100}% 100%`,
                backgroundRepeat: "no-repeat",
              }}
            />
            <span className="text-[13px]">
              <span className="font-medium text-gray-12">{formatMemory(currentStorage)}</span>
            </span>
          </div>
          <SettingDescription>Temporary disk for logs, caches, and scratch data. Changes apply on next deploy.</SettingDescription>
        </div>
      </form>
    </EditableSettingCard>
  );
};

function valueToIndex(mib: number): number {
  const idx = STORAGE_OPTIONS.findIndex((o) => o.value === mib);
  return idx >= 0 ? idx : 0;
}

function indexToValue(index: number): number {
  return STORAGE_OPTIONS[index]?.value ?? 1024;
}

function parseStorageDisplay(mib: number): [string, string] {
  if (mib >= 1024) {
    return [`${(mib / 1024).toFixed(mib % 1024 === 0 ? 0 : 1)}`, "GiB"];
  }
  return [`${mib}`, "MiB"];
}

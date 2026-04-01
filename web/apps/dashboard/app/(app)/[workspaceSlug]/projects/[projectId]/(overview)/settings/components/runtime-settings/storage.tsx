"use client";

import { formatMemoryParts } from "@/lib/utils/deployment-formatters";
import { zodResolver } from "@hookform/resolvers/zod";
import { Harddrive } from "@unkey/icons";
import { Slider } from "@unkey/ui";
import { useForm, useWatch } from "react-hook-form";
import { z } from "zod";
import { SettingDescription, SettingField } from "../shared/form-blocks";
import { FormSettingCard, resolveSaveState } from "../shared/form-setting-card";
import { indexToValue, valueToIndex } from "../shared/resource-slider";

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
    setValue,
    formState: { isValid },
    control,
  } = useForm<StorageFormValues>({
    resolver: zodResolver(storageSchema),
    mode: "onChange",
    defaultValues: { storage: defaultStorage },
  });

  const currentStorage = useWatch({ control, name: "storage" });

  const hasChanges = currentStorage !== defaultStorage;
  const currentIndex = valueToIndex(STORAGE_OPTIONS, currentStorage);

  const saveState = resolveSaveState([
    [!isValid, { status: "disabled" }],
    [!hasChanges, { status: "disabled", reason: "No changes to save" }],
  ]);

  return (
    <FormSettingCard
      icon={<Harddrive className="text-gray-12" iconSize="xl-medium" />}
      title="Storage"
      description="Ephemeral disk space per instance"
      displayValue={(() => {
        const parts = formatMemoryParts(defaultStorage);
        return (
          <div className="space-x-1">
            <span className="font-medium text-gray-12">{parts.value}</span>
            <span className="text-gray-11 font-normal">{parts.unit}</span>
          </div>
        );
      })()}
      onSubmit={(e) => e.preventDefault()}
      saveState={saveState}
    >
      <SettingField>
        <span className="text-gray-11 text-[13px]">Storage per instance</span>
        <div className="flex items-center gap-3">
          <Slider
            min={0}
            max={STORAGE_OPTIONS.length - 1}
            step={1}
            value={[currentIndex]}
            onValueChange={([value]) => {
              if (value !== undefined) {
                setValue("storage", indexToValue(STORAGE_OPTIONS, value, 1024), {
                  shouldValidate: true,
                });
              }
            }}
            className="flex-1 max-w-[var(--setting-w)]"
            rangeStyle={{
              background:
                "linear-gradient(to right, hsla(var(--successA-4)), hsla(var(--successA-12)))",
              backgroundSize: `${currentIndex > 0 ? 100 / (currentIndex / (STORAGE_OPTIONS.length - 1)) : 100}% 100%`,
              backgroundRepeat: "no-repeat",
            }}
          />
          <span className="text-[13px]">
            <span className="font-medium text-gray-12">
              {formatMemoryParts(currentStorage).value}
            </span>{" "}
            <span className="text-gray-11">{formatMemoryParts(currentStorage).unit}</span>
          </span>
        </div>
      </SettingField>
      <SettingDescription>
        Temporary disk for logs, caches, and scratch data. Changes apply on next deploy.
      </SettingDescription>
    </FormSettingCard>
  );
};

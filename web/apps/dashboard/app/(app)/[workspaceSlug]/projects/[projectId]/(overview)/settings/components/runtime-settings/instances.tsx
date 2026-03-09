"use client";

import { collection } from "@/lib/collections";
import { mapRegionToFlag } from "@/lib/trpc/routers/deploy/network/utils";
import { zodResolver } from "@hookform/resolvers/zod";
import { Connections3 } from "@unkey/icons";
import { Slider } from "@unkey/ui";
import { useEffect } from "react";
import { useForm, useWatch } from "react-hook-form";
import { z } from "zod";
import { RegionFlag } from "../../../../components/region-flag";
import { useEnvironmentSettings } from "../../environment-provider";
import { FormSettingCard, resolveSaveState } from "../shared/form-setting-card";
import { SettingDescription } from "../shared/setting-description";

const instancesSchema = z.object({
  instances: z.number().min(1).max(10),
});

type InstancesFormValues = z.infer<typeof instancesSchema>;

export const Instances = () => {
  const { settings, autoSave } = useEnvironmentSettings();
  const { environmentId, regions } = settings;

  const selectedRegions = regions.map((r) => r.name);
  const defaultInstances = regions.at(0)?.replicas ?? 1;

  const {
    handleSubmit,
    setValue,
    formState: { isValid, isSubmitting },
    control,
    reset,
  } = useForm<InstancesFormValues>({
    resolver: zodResolver(instancesSchema),
    mode: "onChange",
    defaultValues: { instances: defaultInstances },
  });

  useEffect(() => {
    reset({ instances: defaultInstances });
  }, [defaultInstances, reset]);

  const currentInstances = useWatch({ control, name: "instances" });

  const onSubmit = async (values: InstancesFormValues) => {
    collection.environmentSettings.update(environmentId, (draft) => {
      for (const region of draft.regions) {
        region.replicas = values.instances;
      }
    });
  };

  const hasChanges = currentInstances !== defaultInstances;
  const hasRegions = regions.length > 0;

  const saveState = resolveSaveState([
    [isSubmitting, { status: "saving" }],
    [
      !hasRegions,
      { status: "disabled", reason: "Select at least one region before setting instance count" },
    ],
    [!isValid, { status: "disabled" }],
    [!hasChanges, { status: "disabled", reason: "No changes to save" }],
  ]);

  return (
    <FormSettingCard
      icon={<Connections3 className="text-gray-12" iconSize="xl-medium" />}
      title="Instances"
      description="Number of instances running in each region"
      displayValue={
        <div className="space-x-1">
          <span className="font-medium text-gray-12">{defaultInstances}</span>
          <span className="text-gray-11 font-normal">
            instance{defaultInstances !== 1 ? "s" : ""}
          </span>
        </div>
      }
      onSubmit={handleSubmit(onSubmit)}
      saveState={saveState}
      autoSave={autoSave}
    >
      <div className="flex flex-col">
        <span className="text-gray-11 text-[13px]">Instances per region</span>
        <div className="flex items-center gap-3">
          <Slider
            min={1}
            max={10}
            step={1}
            value={[currentInstances]}
            onValueChange={([value]) => {
              if (value !== undefined) {
                setValue("instances", value, { shouldValidate: true });
              }
            }}
            onValueCommit={
              autoSave
                ? ([value]) => {
                    if (value !== undefined) {
                      handleSubmit(onSubmit)();
                    }
                  }
                : undefined
            }
            className="flex-1 max-w-[480px]"
            rangeStyle={{
              background:
                "linear-gradient(to right, hsla(var(--featureA-4)), hsla(var(--featureA-12)))",
              backgroundSize: `${currentInstances > 1 ? 100 / ((currentInstances - 1) / 9) : 100}% 100%`,
              backgroundRepeat: "no-repeat",
            }}
          />
          <div className="flex items-center gap-1.5">
            {selectedRegions.map((r) => (
              <RegionFlag
                key={r}
                flagCode={mapRegionToFlag(r)}
                size="xs"
                shape="circle"
                className="[&_img]:size-3"
              />
            ))}
          </div>
          <span className="text-[13px]">
            <span className="font-medium text-gray-12">{currentInstances}</span>{" "}
            <span className="text-gray-11 font-normal">
              instance{currentInstances !== 1 ? "s" : ""}
            </span>
          </span>
        </div>
        <SettingDescription>
          More instances improve availability and handle higher traffic. Changes apply on next
          deploy.
        </SettingDescription>
      </div>
    </FormSettingCard>
  );
};

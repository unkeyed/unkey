"use client";

import { trpc } from "@/lib/trpc/client";
import { mapRegionToFlag } from "@/lib/trpc/routers/deploy/network/utils";
import { zodResolver } from "@hookform/resolvers/zod";
import { Connections3 } from "@unkey/icons";
import { Slider, toast } from "@unkey/ui";
import { useEffect } from "react";
import { useForm, useWatch } from "react-hook-form";
import { z } from "zod";
import { RegionFlag } from "../../../../components/region-flag";
import { useProjectData } from "../../../data-provider";
import { EditableSettingCard } from "../shared/editable-setting-card";

const instancesSchema = z.object({
  instances: z.number().min(1).max(10),
});

type InstancesFormValues = z.infer<typeof instancesSchema>;

export const Instances = () => {
  const { environments } = useProjectData();
  const environmentId = environments[0]?.id;

  const { data: settingsData } = trpc.deploy.environmentSettings.get.useQuery(
    { environmentId: environmentId ?? "" },
    { enabled: Boolean(environmentId) },
  );

  const regionConfig =
    (settingsData?.runtimeSettings?.regionConfig as Record<string, number>) ?? {};
  const selectedRegions = Object.keys(regionConfig);
  const defaultInstances = Object.values(regionConfig)[0] ?? 1;

  return (
    <InstancesForm
      environmentId={environmentId}
      defaultInstances={defaultInstances}
      selectedRegions={selectedRegions}
    />
  );
};

type InstancesFormProps = {
  environmentId: string;
  defaultInstances: number;
  selectedRegions: string[];
};

const InstancesForm: React.FC<InstancesFormProps> = ({
  environmentId,
  defaultInstances,
  selectedRegions,
}) => {
  const utils = trpc.useUtils();

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

  const updateRuntime = trpc.deploy.environmentSettings.updateRuntime.useMutation({
    onSuccess: () => {
      utils.deploy.environmentSettings.get.invalidate({ environmentId });
    },
    onError: (err) => {
      toast.error("Failed to update instances", { description: err.message });
    },
  });

  const onSubmit = async (values: InstancesFormValues) => {
    await updateRuntime.mutateAsync({
      environmentId,
      replicasPerRegion: values.instances,
    });
  };

  const hasChanges = currentInstances !== defaultInstances;

  return (
    <EditableSettingCard
      icon={<Connections3 className="text-gray-12" iconSize="xl-medium" />}
      title="Instances"
      description="Number of instances running in each region"
      border="both"
      displayValue={
        <div className="space-x-1">
          <span className="font-medium text-gray-12">{defaultInstances}</span>
          <span className="text-gray-11 font-normal">
            instance{defaultInstances !== 1 ? "s" : ""}
          </span>
        </div>
      }
      formId="update-instances-form"
      canSave={isValid && !isSubmitting && hasChanges}
      isSaving={updateRuntime.isLoading || isSubmitting}
    >
      <form id="update-instances-form" onSubmit={handleSubmit(onSubmit)}>
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
        </div>
      </form>
    </EditableSettingCard>
  );
};

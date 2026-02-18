"use client";

import { trpc } from "@/lib/trpc/client";
import { formatCpu } from "@/lib/utils/deployment-formatters";
import { zodResolver } from "@hookform/resolvers/zod";
import { Bolt } from "@unkey/icons";
import { Slider, toast } from "@unkey/ui";
import { useEffect } from "react";
import { useForm, useWatch } from "react-hook-form";
import { z } from "zod";
import { useProjectData } from "../../../data-provider";
import { FormSettingCard } from "../shared/form-setting-card";
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

const cpuSchema = z.object({
  cpu: z.number(),
});

type CpuFormValues = z.infer<typeof cpuSchema>;

export const Cpu = () => {
  const { environments } = useProjectData();
  const environmentId = environments[0]?.id;

  const { data: settingsData } = trpc.deploy.environmentSettings.get.useQuery(
    { environmentId: environmentId ?? "" },
    { enabled: Boolean(environmentId) },
  );

  const defaultCpu = settingsData?.runtimeSettings?.cpuMillicores ?? 256;

  return <CpuForm environmentId={environmentId} defaultCpu={defaultCpu} />;
};

type CpuFormProps = {
  environmentId: string;
  defaultCpu: number;
};

const CpuForm: React.FC<CpuFormProps> = ({ environmentId, defaultCpu }) => {
  const utils = trpc.useUtils();

  const {
    handleSubmit,
    setValue,
    formState: { isValid, isSubmitting },
    control,
    reset,
  } = useForm<CpuFormValues>({
    resolver: zodResolver(cpuSchema),
    mode: "onChange",
    defaultValues: { cpu: defaultCpu },
  });

  useEffect(() => {
    reset({ cpu: defaultCpu });
  }, [defaultCpu, reset]);

  const currentCpu = useWatch({ control, name: "cpu" });

  const updateCpu = trpc.deploy.environmentSettings.runtime.updateCpu.useMutation({
    onSuccess: (_data, variables) => {
      toast.success("CPU updated", {
        description: `CPU set to ${formatCpu(variables.cpuMillicores ?? defaultCpu)}`,
        duration: 5000,
      });
      utils.deploy.environmentSettings.get.invalidate({ environmentId });
    },
    onError: (err) => {
      if (err.data?.code === "BAD_REQUEST") {
        toast.error("Invalid CPU setting", {
          description: err.message || "Please check your input and try again.",
        });
      } else {
        toast.error("Failed to update CPU", {
          description:
            err.message ||
            "An unexpected error occurred. Please try again or contact support@unkey.com",
          action: {
            label: "Contact Support",
            onClick: () => window.open("mailto:support@unkey.com", "_blank"),
          },
        });
      }
    },
  });

  const onSubmit = async (values: CpuFormValues) => {
    await updateCpu.mutateAsync({
      environmentId,
      cpuMillicores: values.cpu,
    });
  };

  const hasChanges = currentCpu !== defaultCpu;
  const currentIndex = valueToIndex(CPU_OPTIONS, currentCpu);

  return (
    <FormSettingCard
      icon={<Bolt className="text-gray-12" iconSize="xl-medium" />}
      title="CPU"
      description="CPU allocation for each instance"
      displayValue={(() => {
        const [value, unit] = parseCpuDisplay(defaultCpu);
        return (
          <div className="space-x-1">
            <span className="font-medium text-gray-12">{value}</span>
            <span className="text-gray-11 font-normal">{unit}</span>
          </div>
        );
      })()}
      onSubmit={handleSubmit(onSubmit)}
      canSave={isValid && !isSubmitting && hasChanges}
      isSaving={updateCpu.isLoading || isSubmitting}
    >
      <div className="flex flex-col">
        <span className="text-gray-11 text-[13px]">CPU per instance</span>
        <div className="flex items-center gap-3">
          <Slider
            min={0}
            max={CPU_OPTIONS.length - 1}
            step={1}
            value={[currentIndex]}
            onValueChange={([value]) => {
              if (value !== undefined) {
                setValue("cpu", indexToValue(CPU_OPTIONS, value, 256), { shouldValidate: true });
              }
            }}
            className="flex-1 max-w-[480px]"
            rangeStyle={{
              background: "linear-gradient(to right, hsla(var(--infoA-4)), hsla(var(--infoA-12)))",
              backgroundSize: `${currentIndex > 0 ? 100 / (currentIndex / (CPU_OPTIONS.length - 1)) : 100}% 100%`,
              backgroundRepeat: "no-repeat",
            }}
          />
          <span className="text-[13px]">
            <span className="font-medium text-gray-12">{formatCpu(currentCpu)}</span>
          </span>
        </div>
        <SettingDescription>
          Higher CPU improves compute-heavy workloads. Changes apply on next deploy.
        </SettingDescription>
      </div>
    </FormSettingCard>
  );
};

function parseCpuDisplay(millicores: number): [string, string] {
  if (millicores === 256) {
    return ["1/4", "vCPU"];
  }
  if (millicores === 512) {
    return ["1/2", "vCPU"];
  }
  if (millicores === 768) {
    return ["3/4", "vCPU"];
  }
  if (millicores >= 1024 && millicores % 1024 === 0) {
    return [`${millicores / 1024}`, "vCPU"];
  }
  return [`${millicores}m`, "vCPU"];
}

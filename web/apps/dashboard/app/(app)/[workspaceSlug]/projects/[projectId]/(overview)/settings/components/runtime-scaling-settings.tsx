"use client";

import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import {
  Button,
  FormInput,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  SettingCard,
  toast,
} from "@unkey/ui";
import { useForm, useWatch } from "react-hook-form";
import { z } from "zod";

type Props = {
  environmentId: string;
};

const CPU_OPTIONS = [
  { label: "0.25 vCPU", value: 256, disabled: false },
  { label: "0.5 vCPU", value: 512, disabled: false },
  { label: "1 vCPU", value: 1024, disabled: false },
  { label: "2 vCPU", value: 2048, disabled: false },
  { label: "4 vCPU", value: 4096, disabled: false },
  { label: "8 vCPU", value: 8192, disabled: true },
  { label: "16 vCPU", value: 16384, disabled: true },
  { label: "32 vCPU", value: 32768, disabled: true },
] as const;

const MEMORY_OPTIONS = [
  { label: "256 MB", value: 256, disabled: false },
  { label: "512 MB", value: 512, disabled: false },
  { label: "1 GB", value: 1024, disabled: false },
  { label: "2 GB", value: 2048, disabled: false },
  { label: "4 GB", value: 4096, disabled: false },
  { label: "8 GB", value: 8192, disabled: true },
  { label: "16 GB", value: 16384, disabled: true },
  { label: "32 GB", value: 32768, disabled: true },
] as const;

const replicasSchema = z.object({
  replicas: z.number().min(1).max(10),
});

const cpuSchema = z.object({ cpu: z.number() });
const memorySchema = z.object({ memory: z.number() });

const CpuCard: React.FC<Props & { defaultCpu: number }> = ({ environmentId, defaultCpu }) => {
  const utils = trpc.useUtils();

  const {
    handleSubmit,
    formState: { isValid, isSubmitting },
    setValue,
    control,
  } = useForm<z.infer<typeof cpuSchema>>({
    resolver: zodResolver(cpuSchema),
    mode: "onChange",
    defaultValues: { cpu: defaultCpu },
  });

  const currentCpu = useWatch({ control, name: "cpu" });

  const updateRuntime = trpc.deploy.environmentSettings.updateRuntime.useMutation({
    onSuccess: () => {
      toast.success("CPU updated");
      utils.deploy.environmentSettings.get.invalidate({ environmentId });
    },
    onError: (err) => {
      toast.error("Failed to update CPU", { description: err.message });
    },
  });

  const onSubmit = async (values: z.infer<typeof cpuSchema>) => {
    await updateRuntime.mutateAsync({
      environmentId,
      cpuMillicores: values.cpu,
    });
  };

  return (
    <form onSubmit={handleSubmit(onSubmit)}>
      <SettingCard
        title="CPU"
        description="The amount of CPU allocated to each replica."
        border="top"
        contentWidth="w-full lg:w-[420px]"
      >
        <div className="flex flex-row justify-end items-center w-full gap-x-2">
          <Select
            value={String(currentCpu)}
            onValueChange={(v) => {
              setValue("cpu", Number(v), { shouldValidate: true });
            }}
          >
            <SelectTrigger>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {CPU_OPTIONS.map((opt) => (
                <SelectItem key={opt.value} value={String(opt.value)} disabled={opt.disabled}>
                  {opt.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          <Button
            type="submit"
            variant="primary"
            size="lg"
            disabled={
              updateRuntime.isLoading || isSubmitting || !isValid || currentCpu === defaultCpu
            }
            loading={updateRuntime.isLoading || isSubmitting}
          >
            Save
          </Button>
        </div>
      </SettingCard>
    </form>
  );
};

const MemoryCard: React.FC<Props & { defaultMemory: number }> = ({
  environmentId,
  defaultMemory,
}) => {
  const utils = trpc.useUtils();

  const {
    handleSubmit,
    formState: { isValid, isSubmitting },
    setValue,
    control,
  } = useForm<z.infer<typeof memorySchema>>({
    resolver: zodResolver(memorySchema),
    mode: "onChange",
    defaultValues: { memory: defaultMemory },
  });

  const currentMemory = useWatch({ control, name: "memory" });

  const updateRuntime = trpc.deploy.environmentSettings.updateRuntime.useMutation({
    onSuccess: () => {
      toast.success("Memory updated");
      utils.deploy.environmentSettings.get.invalidate({ environmentId });
    },
    onError: (err) => {
      toast.error("Failed to update memory", { description: err.message });
    },
  });

  const onSubmit = async (values: z.infer<typeof memorySchema>) => {
    await updateRuntime.mutateAsync({
      environmentId,
      memoryMib: values.memory,
    });
  };

  return (
    <form onSubmit={handleSubmit(onSubmit)}>
      <SettingCard
        title="Memory"
        description="The amount of memory allocated to each replica."
        border="default"
        contentWidth="w-full lg:w-[420px]"
      >
        <div className="flex flex-row justify-end items-center w-full gap-x-2">
          <Select
            value={String(currentMemory)}
            onValueChange={(v) => {
              setValue("memory", Number(v), { shouldValidate: true });
            }}
          >
            <SelectTrigger>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {MEMORY_OPTIONS.map((opt) => (
                <SelectItem key={opt.value} value={String(opt.value)} disabled={opt.disabled}>
                  {opt.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          <Button
            type="submit"
            variant="primary"
            size="lg"
            disabled={
              updateRuntime.isLoading || isSubmitting || !isValid || currentMemory === defaultMemory
            }
            loading={updateRuntime.isLoading || isSubmitting}
          >
            Save
          </Button>
        </div>
      </SettingCard>
    </form>
  );
};

const ReplicasCard: React.FC<Props & { defaultReplicas: number }> = ({
  environmentId,
  defaultReplicas,
}) => {
  const utils = trpc.useUtils();

  const {
    register,
    handleSubmit,
    formState: { errors, isValid, isSubmitting },
    control,
  } = useForm<z.infer<typeof replicasSchema>>({
    resolver: zodResolver(replicasSchema),
    mode: "onChange",
    defaultValues: { replicas: defaultReplicas },
  });

  const currentReplicas = useWatch({ control, name: "replicas" });

  const updateRuntime = trpc.deploy.environmentSettings.updateRuntime.useMutation({
    onSuccess: () => {
      toast.success("Replicas updated");
      utils.deploy.environmentSettings.get.invalidate({ environmentId });
    },
    onError: (err) => {
      toast.error("Failed to update replicas", {
        description: err.message,
      });
    },
  });

  const onSubmit = async (values: z.infer<typeof replicasSchema>) => {
    await updateRuntime.mutateAsync({
      environmentId,
      replicasPerRegion: values.replicas,
    });
  };

  return (
    <form onSubmit={handleSubmit(onSubmit)}>
      <SettingCard
        title="Replicas per Region"
        description="Number of replicas to run in each region."
        border="bottom"
        contentWidth="w-full lg:w-[420px]"
      >
        <div className="flex flex-row justify-end items-center w-full gap-x-2">
          <FormInput
            className="w-full"
            type="number"
            min={1}
            max={10}
            placeholder="1"
            error={errors.replicas?.message}
            {...register("replicas", { valueAsNumber: true })}
          />
          <Button
            type="submit"
            variant="primary"
            size="lg"
            disabled={
              updateRuntime.isLoading ||
              isSubmitting ||
              !isValid ||
              currentReplicas === defaultReplicas
            }
            loading={updateRuntime.isLoading || isSubmitting}
          >
            Save
          </Button>
        </div>
      </SettingCard>
    </form>
  );
};

export const RuntimeScalingSettings: React.FC<Props> = ({ environmentId }) => {
  const { data } = trpc.deploy.environmentSettings.get.useQuery({ environmentId });
  const runtimeSettings = data?.runtimeSettings;

  return (
    <div>
      <CpuCard environmentId={environmentId} defaultCpu={runtimeSettings?.cpuMillicores ?? 1000} />
      <MemoryCard
        environmentId={environmentId}
        defaultMemory={runtimeSettings?.memoryMib ?? 1024}
      />
      <ReplicasCard
        environmentId={environmentId}
        defaultReplicas={
          Object.values((runtimeSettings?.regionConfig as Record<string, number>) ?? {})[0] ?? 1
        }
      />
    </div>
  );
};

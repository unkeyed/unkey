import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import { NumberInput } from "@unkey/icons";
import { FormInput, toast } from "@unkey/ui";
import { useForm, useWatch } from "react-hook-form";
import { z } from "zod";
import { useProjectData } from "../../../data-provider";
import { FormSettingCard } from "../shared/form-setting-card";

const portSchema = z.object({
  port: z.number().int().min(1).max(65535),
});

export const PortSettings = () => {
  const { environments } = useProjectData();
  const environmentId = environments[0]?.id;

  const { data } = trpc.deploy.environmentSettings.get.useQuery(
    { environmentId: environmentId ?? "" },
    { enabled: Boolean(environmentId) },
  );

  const defaultValue = data?.runtimeSettings?.port ?? 8080;
  return <PortForm environmentId={environmentId} defaultValue={defaultValue} />;
};

const PortForm = ({
  environmentId,
  defaultValue,
}: {
  environmentId: string | undefined;
  defaultValue: number;
}) => {
  const utils = trpc.useUtils();

  const {
    register,
    handleSubmit,
    formState: { isValid, isSubmitting, errors },
    control,
  } = useForm<z.infer<typeof portSchema>>({
    resolver: zodResolver(portSchema),
    mode: "onChange",
    defaultValues: { port: defaultValue },
  });

  const currentPort = useWatch({ control, name: "port" });

  const updateRuntime = trpc.deploy.environmentSettings.updateRuntime.useMutation({
    onSuccess: (_data, variables) => {
      toast.success("Port updated", {
        description: `Port set to ${variables.port ?? defaultValue}.`,
        duration: 5000,
      });
      utils.deploy.environmentSettings.get.invalidate({ environmentId });
    },
    onError: (err) => {
      if (err.data?.code === "BAD_REQUEST") {
        toast.error("Invalid port", {
          description: err.message || "Please check your input and try again.",
        });
      } else {
        toast.error("Failed to update port", {
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

  const onSubmit = async (values: z.infer<typeof portSchema>) => {
    await updateRuntime.mutateAsync({ environmentId: environmentId ?? "", port: values.port });
  };

  return (
    <FormSettingCard
      icon={<NumberInput className="text-gray-12" iconSize="xl-medium" />}
      title="Port"
      border="bottom"
      description="Port your application listens on"
      displayValue={String(defaultValue)}
      onSubmit={handleSubmit(onSubmit)}
      canSave={isValid && !isSubmitting && currentPort !== defaultValue}
      isSaving={updateRuntime.isLoading || isSubmitting}
    >
      <FormInput
        required
        type="number"
        className="w-[480px]"
        label="Port"
        description="Port your application listens on. Changes apply on next deploy."
        placeholder="8080"
        min={2000}
        max={54000}
        error={errors.port?.message}
        variant={errors.port ? "error" : "default"}
        {...register("port", { valueAsNumber: true })}
      />
    </FormSettingCard>
  );
};

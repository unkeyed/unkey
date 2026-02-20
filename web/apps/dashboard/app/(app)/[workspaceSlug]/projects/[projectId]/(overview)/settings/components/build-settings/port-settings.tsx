import { collection } from "@/lib/collections";
import { zodResolver } from "@hookform/resolvers/zod";
import { eq, useLiveQuery } from "@tanstack/react-db";
import { NumberInput } from "@unkey/icons";
import { FormInput } from "@unkey/ui";
import { useForm, useWatch } from "react-hook-form";
import { z } from "zod";
import { useEnvironmentId } from "../../environment-provider";
import { FormSettingCard } from "../shared/form-setting-card";

const portSchema = z.object({
  port: z.number().int().min(2000).max(54000),
});

export const PortSettings = () => {
  const environmentId = useEnvironmentId();

  const { data: settings } = useLiveQuery(
    (q) =>
      q
        .from({ s: collection.environmentSettings })
        .where(({ s }) => eq(s.environmentId, environmentId)),
    [environmentId],
  );

  const defaultValue = settings?.[0]?.port ?? 8080;
  return <PortForm environmentId={environmentId} defaultValue={defaultValue} />;
};

const PortForm = ({
  environmentId,
  defaultValue,
}: {
  environmentId: string;
  defaultValue: number;
}) => {
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

  const onSubmit = async (values: z.infer<typeof portSchema>) => {
    collection.environmentSettings.update(environmentId, (draft) => {
      draft.port = values.port;
    });
  };

  return (
    <FormSettingCard
      icon={<NumberInput className="text-gray-12" iconSize="xl-medium" />}
      title="Port"
      description="Port your application listens on"
      displayValue={String(defaultValue)}
      onSubmit={handleSubmit(onSubmit)}
      canSave={isValid && !isSubmitting && currentPort !== defaultValue}
      isSaving={isSubmitting}
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

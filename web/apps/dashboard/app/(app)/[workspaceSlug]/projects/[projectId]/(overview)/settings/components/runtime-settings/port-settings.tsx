import { collection } from "@/lib/collections";
import { zodResolver } from "@hookform/resolvers/zod";
import { NumberInput } from "@unkey/icons";
import { FormInput } from "@unkey/ui";
import { useForm, useWatch } from "react-hook-form";
import { z } from "zod";
import { useEnvironmentSettings } from "../../environment-provider";
import { FormSettingCard } from "../shared/form-setting-card";

const portSchema = z.object({
  port: z.number().int().min(2000).max(54000),
});

export const Port = () => {
  const { settings } = useEnvironmentSettings();
  const { environmentId, port: defaultValue } = settings;

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

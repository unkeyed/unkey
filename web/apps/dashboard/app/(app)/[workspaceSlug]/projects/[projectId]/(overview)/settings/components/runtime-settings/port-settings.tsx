import { zodResolver } from "@hookform/resolvers/zod";
import { NumberInput } from "@unkey/icons";
import { FormInput } from "@unkey/ui";
import { useForm, useWatch } from "react-hook-form";
import { z } from "zod";
import { useEnvironmentSettings } from "../../environment-provider";
import { useUpdateAllEnvironments } from "../../hooks/use-update-all-environments";
import { SettingField } from "../shared/form-blocks";
import { FormSettingCard, resolveSaveState } from "../shared/form-setting-card";

const portSchema = z.object({
  port: z.number().int().min(1).max(65535),
});

export const Port = () => {
  const { settings, variant } = useEnvironmentSettings();
  const { port: defaultValue } = settings;
  const updateAllEnvironments = useUpdateAllEnvironments();

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

  const saveState = resolveSaveState([
    [isSubmitting, { status: "saving" }],
    [!isValid, { status: "disabled" }],
    [currentPort === defaultValue, { status: "disabled", reason: "No changes to save" }],
  ]);

  const onSubmit = async (values: z.infer<typeof portSchema>) => {
    updateAllEnvironments((draft) => {
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
      saveState={saveState}
      autoSave={variant === "onboarding"}
    >
      <SettingField>
        <FormInput
          required
          type="number"
          onWheelCapture={(e) => {
            //@ts-expect-error there is no other way to prevent scroll here
            e.target.blur();
          }}
          label="Port"
          description="Port your application listens on. Changes apply on next deploy."
          placeholder="8080"
          min={1}
          max={65535}
          error={errors.port?.message}
          variant={errors.port ? "error" : "default"}
          {...register("port", { valueAsNumber: true })}
        />
      </SettingField>
    </FormSettingCard>
  );
};

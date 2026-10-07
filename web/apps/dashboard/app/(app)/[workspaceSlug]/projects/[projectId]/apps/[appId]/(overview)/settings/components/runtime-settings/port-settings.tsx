import { zodResolver } from "@hookform/resolvers/zod";
import { FormInput, formSaveState } from "@unkey/ui";
import { useForm, useWatch } from "react-hook-form";
import { z } from "zod";
import { useEnvironmentSettings } from "../../environment-provider";
import { useUpdateAllEnvironments } from "../../hooks/use-update-all-environments";
import { SettingField } from "../shared/form-blocks";
import { FormSettingCard } from "../shared/form-setting-card";

const portSchema = z.object({
  port: z.number().int().min(1).max(65535),
});

export const Port = () => {
  const { settings, autoSave } = useEnvironmentSettings();
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
    values: { port: defaultValue },
  });

  const currentPort = useWatch({ control, name: "port" });

  const dirty = currentPort !== defaultValue;
  const saveState = formSaveState({
    isSubmitting,
    isValid,
    isDirty: dirty,
  });

  const onSubmit = async (values: z.infer<typeof portSchema>) => {
    updateAllEnvironments((draft) => {
      draft.port = values.port;
    });
  };

  return (
    <FormSettingCard
      title="Port"
      onSubmit={handleSubmit(onSubmit)}
      dirty={dirty}
      saveState={saveState}
      autoSave={autoSave}
    >
      <SettingField>
        <FormInput
          data-1p-ignore
          type="number"
          onWheelCapture={(e) => {
            //@ts-expect-error there is no other way to prevent scroll here
            e.target.blur();
          }}
          aria-label="Port"
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

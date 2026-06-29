import { zodResolver } from "@hookform/resolvers/zod";
import { Cube } from "@unkey/icons";
import { FormInput } from "@unkey/ui";
import { useForm, useWatch } from "react-hook-form";
import { z } from "zod";
import { useEnvironmentSettings } from "../../environment-provider";
import { useUpdateAllEnvironments } from "../../hooks/use-update-all-environments";
import { SettingField } from "../shared/form-blocks";
import { FormSettingCard, resolveSaveState } from "../shared/form-setting-card";

const installCommandSchema = z.object({
  // Empty means "let Railpack auto-detect the install command".
  installCommand: z.string().max(1000),
});

export const InstallCommand = () => {
  const { settings, variant } = useEnvironmentSettings();
  const { installCommand: defaultValue } = settings;
  const updateAllEnvironments = useUpdateAllEnvironments();

  const {
    register,
    handleSubmit,
    formState: { isValid, isSubmitting, errors },
    control,
  } = useForm<z.infer<typeof installCommandSchema>>({
    resolver: zodResolver(installCommandSchema),
    mode: "onChange",
    defaultValues: { installCommand: defaultValue },
  });

  const currentInstallCommand = useWatch({ control, name: "installCommand" });

  const saveState = resolveSaveState([
    [isSubmitting, { status: "saving" }],
    [!isValid, { status: "disabled" }],
    [currentInstallCommand === defaultValue, { status: "disabled", reason: "No changes to save" }],
  ]);

  const onSubmit = async (values: z.infer<typeof installCommandSchema>) => {
    updateAllEnvironments((draft) => {
      draft.installCommand = values.installCommand;
    });
  };

  return (
    <FormSettingCard
      icon={<Cube className="text-gray-12" iconSize="xl-medium" />}
      title="Install command"
      description="Override the auto-detected install command. Useful for monorepos, e.g. pnpm install --filter api. Only applies when no Dockerfile is set."
      displayValue={defaultValue || "Automatic"}
      onSubmit={handleSubmit(onSubmit)}
      saveState={saveState}
      autoSave={variant === "onboarding"}
    >
      <SettingField>
        <FormInput
          label="Install command"
          requirement="optional"
          description="Leave empty to let Unkey detect it automatically. Changes apply on next deploy."
          placeholder="Automatic"
          error={errors.installCommand?.message}
          {...register("installCommand")}
        />
      </SettingField>
    </FormSettingCard>
  );
};

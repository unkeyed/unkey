import { zodResolver } from "@hookform/resolvers/zod";
import { Hammer2 } from "@unkey/icons";
import { FormInput } from "@unkey/ui";
import { useForm, useWatch } from "react-hook-form";
import { z } from "zod";
import { useEnvironmentSettings } from "../../environment-provider";
import { useUpdateAllEnvironments } from "../../hooks/use-update-all-environments";
import { SettingField } from "../shared/form-blocks";
import { FormSettingCard, resolveSaveState } from "../shared/form-setting-card";

const buildCommandSchema = z.object({
  // Empty means "let Railpack auto-detect the build command".
  buildCommand: z.string().max(1000),
});

export const BuildCommand = () => {
  const { settings, variant } = useEnvironmentSettings();
  const { buildCommand: defaultValue } = settings;
  const updateAllEnvironments = useUpdateAllEnvironments();

  const {
    register,
    handleSubmit,
    formState: { isValid, isSubmitting, errors },
    control,
  } = useForm<z.infer<typeof buildCommandSchema>>({
    resolver: zodResolver(buildCommandSchema),
    mode: "onChange",
    defaultValues: { buildCommand: defaultValue },
  });

  const currentBuildCommand = useWatch({ control, name: "buildCommand" });

  const saveState = resolveSaveState([
    [isSubmitting, { status: "saving" }],
    [!isValid, { status: "disabled" }],
    [currentBuildCommand === defaultValue, { status: "disabled", reason: "No changes to save" }],
  ]);

  const onSubmit = async (values: z.infer<typeof buildCommandSchema>) => {
    updateAllEnvironments((draft) => {
      draft.buildCommand = values.buildCommand;
    });
  };

  return (
    <FormSettingCard
      icon={<Hammer2 className="text-gray-12" iconSize="xl-medium" />}
      title="Build command"
      description="Override the auto-detected build command. Useful for monorepos, e.g. pnpm build --filter api. Only applies when no Dockerfile is set."
      displayValue={defaultValue || "Automatic"}
      onSubmit={handleSubmit(onSubmit)}
      saveState={saveState}
      autoSave={variant === "onboarding"}
    >
      <SettingField>
        <FormInput
          label="Build command"
          requirement="optional"
          description="Leave empty to let Unkey detect it automatically. Changes apply on next deploy."
          placeholder="Automatic"
          error={errors.buildCommand?.message}
          {...register("buildCommand")}
        />
      </SettingField>
    </FormSettingCard>
  );
};

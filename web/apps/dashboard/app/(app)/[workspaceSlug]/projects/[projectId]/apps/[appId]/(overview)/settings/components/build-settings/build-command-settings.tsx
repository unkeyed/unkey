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
  const { buildCommand: defaultValue, dockerfile } = settings;
  const updateAllEnvironments = useUpdateAllEnvironments();

  // The build command only applies to automatic (Railpack) builds. When a
  // Dockerfile is configured the build follows the Dockerfile and Railpack is
  // never invoked, so the command is ignored — disable the field to make that
  // explicit. settings.dockerfile is reactive: editing the Dockerfile card
  // re-renders this one.
  const dockerfileConfigured = dockerfile.trim() !== "";

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
    [
      dockerfileConfigured,
      {
        status: "disabled",
        reason: "Not used with a Dockerfile build. Remove the Dockerfile to build automatically.",
      },
    ],
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
      displayValue={
        dockerfileConfigured ? "Not used (Dockerfile build)" : defaultValue || "Automatic"
      }
      onSubmit={handleSubmit(onSubmit)}
      saveState={saveState}
      autoSave={variant === "onboarding"}
    >
      <SettingField>
        <FormInput
          label="Build command"
          requirement="optional"
          description={
            dockerfileConfigured
              ? "Disabled because a Dockerfile is configured. Build commands only apply to automatic (Railpack) builds."
              : "Leave empty to let Unkey detect it automatically. Changes apply on next deploy."
          }
          placeholder="Automatic"
          disabled={dockerfileConfigured}
          error={errors.buildCommand?.message}
          {...register("buildCommand")}
        />
      </SettingField>
    </FormSettingCard>
  );
};

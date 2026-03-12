import { zodResolver } from "@hookform/resolvers/zod";
import { FolderLink } from "@unkey/icons";
import { FormInput } from "@unkey/ui";
import { useForm, useWatch } from "react-hook-form";
import { z } from "zod";
import { useEnvironmentSettings } from "../../environment-provider";
import { useUpdateAllEnvironments } from "../../hooks/use-update-all-environments";
import { FormSettingCard, resolveSaveState } from "../shared/form-setting-card";

const rootDirectorySchema = z.object({
  dockerContext: z.string(),
});

export const RootDirectory = () => {
  const { settings, variant } = useEnvironmentSettings();
  const { dockerContext: defaultValue } = settings;
  const updateAllEnvironments = useUpdateAllEnvironments();

  const {
    register,
    handleSubmit,
    formState: { isValid, isSubmitting, errors },
    control,
  } = useForm<z.infer<typeof rootDirectorySchema>>({
    resolver: zodResolver(rootDirectorySchema),
    mode: "onChange",
    defaultValues: { dockerContext: defaultValue },
  });

  const currentDockerContext = useWatch({ control, name: "dockerContext" });

  const saveState = resolveSaveState([
    [isSubmitting, { status: "saving" }],
    [!isValid, { status: "disabled" }],
    [currentDockerContext === defaultValue, { status: "disabled", reason: "No changes to save" }],
  ]);

  const onSubmit = async (values: z.infer<typeof rootDirectorySchema>) => {
    updateAllEnvironments((draft) => {
      draft.dockerContext = values.dockerContext;
    });
  };

  return (
    <FormSettingCard
      icon={<FolderLink className="text-gray-12" iconSize="xl-medium" />}
      title="Root directory"
      description="Build context directory. All COPY/ADD commands are relative to this path. (e.g., services/api)"
      displayValue={defaultValue || "."}
      onSubmit={handleSubmit(onSubmit)}
      saveState={saveState}
      autoSave={variant === "onboarding"}
    >
      <FormInput
        label="Root directory"
        required
        className="w-[480px]"
        description="Build context directory for Docker. Changes apply on next deploy."
        placeholder="."
        error={errors.dockerContext?.message}
        variant={errors.dockerContext ? "error" : "default"}
        {...register("dockerContext")}
      />
    </FormSettingCard>
  );
};

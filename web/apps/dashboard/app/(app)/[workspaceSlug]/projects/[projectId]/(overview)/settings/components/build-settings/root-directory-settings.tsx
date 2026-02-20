import { collection } from "@/lib/collections";
import { zodResolver } from "@hookform/resolvers/zod";
import { FolderLink } from "@unkey/icons";
import { FormInput } from "@unkey/ui";
import { useForm, useWatch } from "react-hook-form";
import { z } from "zod";
import { useEnvironmentSettings } from "../../environment-provider";
import { FormSettingCard } from "../shared/form-setting-card";

const rootDirectorySchema = z.object({
  dockerContext: z.string(),
});

export const RootDirectory = () => {
  const { settings } = useEnvironmentSettings();
  const { environmentId, dockerContext: defaultValue } = settings;

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

  const onSubmit = async (values: z.infer<typeof rootDirectorySchema>) => {
    collection.environmentSettings.update(environmentId, (draft) => {
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
      canSave={isValid && !isSubmitting && currentDockerContext !== defaultValue}
      isSaving={isSubmitting}
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

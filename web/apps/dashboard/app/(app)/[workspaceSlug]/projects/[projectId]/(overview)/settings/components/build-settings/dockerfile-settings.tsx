import { collection } from "@/lib/collections";
import { zodResolver } from "@hookform/resolvers/zod";
import { FileSettings } from "@unkey/icons";
import { FormInput } from "@unkey/ui";
import { useForm, useWatch } from "react-hook-form";
import { z } from "zod";
import { useEnvironmentSettings } from "../../environment-provider";
import { FormSettingCard } from "../shared/form-setting-card";

const dockerfileSchema = z.object({
  dockerfile: z.string().min(1, "Dockerfile path is required"),
});

export const Dockerfile = () => {
  const { settings } = useEnvironmentSettings();
  const { environmentId, dockerfile: defaultValue } = settings;

  const {
    register,
    handleSubmit,
    formState: { isValid, isSubmitting, errors },
    control,
  } = useForm<z.infer<typeof dockerfileSchema>>({
    resolver: zodResolver(dockerfileSchema),
    mode: "onChange",
    defaultValues: { dockerfile: defaultValue },
  });

  const currentDockerfile = useWatch({ control, name: "dockerfile" });

  const onSubmit = async (values: z.infer<typeof dockerfileSchema>) => {
    collection.environmentSettings.update(environmentId, (draft) => {
      draft.dockerfile = values.dockerfile;
    });
  };

  return (
    <FormSettingCard
      icon={<FileSettings className="text-gray-12" iconSize="xl-medium" />}
      title="Dockerfile"
      description="Dockerfile location used for docker build. (e.g., services/api/Dockerfile)"
      displayValue={defaultValue}
      onSubmit={handleSubmit(onSubmit)}
      canSave={isValid && !isSubmitting && currentDockerfile !== defaultValue}
      isSaving={isSubmitting}
    >
      <FormInput
        required
        className="w-[480px]"
        label="Dockerfile path"
        description="Dockerfile location used for docker build. Changes apply on next deploy."
        placeholder="Dockerfile"
        error={errors.dockerfile?.message}
        variant={errors.dockerfile ? "error" : "default"}
        {...register("dockerfile")}
      />
    </FormSettingCard>
  );
};

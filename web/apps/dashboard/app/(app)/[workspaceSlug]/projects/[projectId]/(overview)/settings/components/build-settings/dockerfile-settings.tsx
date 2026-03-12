import { zodResolver } from "@hookform/resolvers/zod";
import { FileSettings } from "@unkey/icons";
import { FormInput } from "@unkey/ui";
import { useForm, useWatch } from "react-hook-form";
import { z } from "zod";
import { useEnvironmentSettings } from "../../environment-provider";
import { useUpdateAllEnvironments } from "../../hooks/use-update-all-environments";
import { FormSettingCard, resolveSaveState } from "../shared/form-setting-card";

const dockerfileSchema = z.object({
  dockerfile: z.string().min(1, "Dockerfile path is required"),
});

export const Dockerfile = () => {
  const { settings, variant } = useEnvironmentSettings();
  const { dockerfile: defaultValue } = settings;
  const updateAllEnvironments = useUpdateAllEnvironments();

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

  const saveState = resolveSaveState([
    [isSubmitting, { status: "saving" }],
    [!isValid, { status: "disabled" }],
    [currentDockerfile === defaultValue, { status: "disabled", reason: "No changes to save" }],
  ]);

  const onSubmit = async (values: z.infer<typeof dockerfileSchema>) => {
    updateAllEnvironments((draft) => {
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
      saveState={saveState}
      autoSave={variant === "onboarding"}
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

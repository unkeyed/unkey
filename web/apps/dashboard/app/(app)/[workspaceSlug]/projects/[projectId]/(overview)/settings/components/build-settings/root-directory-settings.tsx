import { zodResolver } from "@hookform/resolvers/zod";
import { FolderLink } from "@unkey/icons";
import { FormInput } from "@unkey/ui";
import { useForm, useWatch } from "react-hook-form";
import { z } from "zod";
import { useEnvironmentSettings } from "../../environment-provider";
import { useUpdateAllEnvironments } from "../../hooks/use-update-all-environments";
import { FormSettingCard, resolveSaveState } from "../shared/form-setting-card";
import { useRepoTree } from "./use-repo-tree";

const rootDirectorySchema = z.object({
  dockerContext: z.string(),
});

export const RootDirectory = () => {
  const { settings, variant } = useEnvironmentSettings();
  const { dockerContext: defaultValue } = settings;
  const updateAllEnvironments = useUpdateAllEnvironments();
  const { validatePath, findCaseInsensitiveMatch } = useRepoTree();

  const {
    register,
    handleSubmit,
    formState: { isValid, isSubmitting, errors },
    control,
    setValue,
  } = useForm<z.infer<typeof rootDirectorySchema>>({
    resolver: zodResolver(rootDirectorySchema),
    mode: "onChange",
    defaultValues: { dockerContext: defaultValue },
  });

  const currentDockerContext = useWatch({ control, name: "dockerContext" });

  const validation = validatePath(currentDockerContext, "tree");
  const caseMatch =
    validation === "invalid" ? findCaseInsensitiveMatch(currentDockerContext, "tree") : null;

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

  const inputVariant = errors.dockerContext
    ? "error"
    : validation === "invalid"
      ? "warning"
      : "default";

  const warningMessage =
    validation === "invalid" ? (
      caseMatch ? (
        <span>
          Did you mean{" "}
          <button
            type="button"
            className="underline font-medium hover:text-warning-12"
            onClick={() => setValue("dockerContext", caseMatch, { shouldValidate: true })}
          >
            {caseMatch}
          </button>
          ?
        </span>
      ) : (
        "This directory was not found in the connected repository"
      )
    ) : undefined;

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
        description={
          warningMessage ?? "Build context directory for Docker. Changes apply on next deploy."
        }
        placeholder="."
        error={errors.dockerContext?.message}
        variant={inputVariant}
        {...register("dockerContext")}
      />
    </FormSettingCard>
  );
};

import { FormCombobox } from "@/components/ui/form-combobox";
import { zodResolver } from "@hookform/resolvers/zod";
import { FileSettings } from "@unkey/icons";
import { useMemo } from "react";
import { useForm, useWatch } from "react-hook-form";
import { z } from "zod";
import { useEnvironmentSettings } from "../../environment-provider";
import { useUpdateAllEnvironments } from "../../hooks/use-update-all-environments";
import { SettingDescription, SettingField } from "../shared/form-blocks";
import { FormSettingCard, resolveSaveState } from "../shared/form-setting-card";
import { useRepoTree } from "./use-repo-tree";

const dockerfileSchema = z.object({
  dockerfile: z.string().min(1, "Dockerfile path is required"),
});

export const Dockerfile = () => {
  const { settings, variant } = useEnvironmentSettings();
  const { dockerfile: defaultValue, dockerContext } = settings;
  const updateAllEnvironments = useUpdateAllEnvironments();
  const { validateDockerfilePath, findDockerfileCaseMatch, getDockerfilesForContext } =
    useRepoTree();

  const {
    handleSubmit,
    formState: { isValid, isSubmitting, errors },
    control,
    setValue,
  } = useForm<z.infer<typeof dockerfileSchema>>({
    resolver: zodResolver(dockerfileSchema),
    mode: "onChange",
    defaultValues: { dockerfile: defaultValue },
  });

  const currentDockerfile = useWatch({ control, name: "dockerfile", defaultValue });

  const validation = validateDockerfilePath(currentDockerfile, dockerContext);
  const caseMatch =
    validation === "invalid" ? findDockerfileCaseMatch(currentDockerfile, dockerContext) : null;
  const detectedDockerfiles = getDockerfilesForContext(dockerContext);

  const options = useMemo(
    () => detectedDockerfiles.map((path) => ({ label: path, value: path })),
    [detectedDockerfiles],
  );

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

  const inputVariant = errors.dockerfile
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
            onClick={() => setValue("dockerfile", caseMatch, { shouldValidate: true })}
          >
            {caseMatch}
          </button>
          ?
        </span>
      ) : (
        "This file was not found in the connected repository"
      )
    ) : undefined;

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
      <SettingField>
        <FormCombobox
          required
          label="Dockerfile"
          options={options}
          value={currentDockerfile}
          onSelect={(val) => setValue("dockerfile", val, { shouldValidate: true })}
          creatable
          searchPlaceholder="Search or type a path..."
          emptyMessage={<div className="mt-2">No Dockerfiles detected in repository</div>}
          placeholder={<span className="text-grayA-8">Dockerfile</span>}
          variant={inputVariant}
        />
      </SettingField>

      {warningMessage ? (
        <div className="text-[13px] leading-5 text-warning-11">{warningMessage}</div>
      ) : (
        <SettingDescription>
          Dockerfile location used for docker build. Changes apply on next deploy.
        </SettingDescription>
      )}
    </FormSettingCard>
  );
};

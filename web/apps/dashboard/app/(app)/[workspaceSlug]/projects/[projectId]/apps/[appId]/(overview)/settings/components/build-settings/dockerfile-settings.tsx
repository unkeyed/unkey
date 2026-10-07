import { FormCombobox } from "@/components/ui/form-combobox";
import { zodResolver } from "@hookform/resolvers/zod";
import { formSaveState } from "@unkey/ui";
import { useMemo } from "react";
import { useForm, useWatch } from "react-hook-form";
import { z } from "zod";
import { useEnvironmentSettings } from "../../environment-provider";
import { useUpdateAllEnvironments } from "../../hooks/use-update-all-environments";
import { SettingField } from "../shared/form-blocks";
import { FormSettingCard } from "../shared/form-setting-card";
import { pathHint, pathInputVariant } from "./path-hint";
import { pathHintMessage } from "./path-hint-message";
import { useRepoTree } from "./use-repo-tree";

const dockerfileSchema = z.object({
  // Empty means "no Dockerfile configured": the app builds with Railpack.
  dockerfile: z.string(),
});

export const Dockerfile = () => {
  const { settings, autoSave } = useEnvironmentSettings();
  const { dockerfile: defaultValue, dockerContext } = settings;
  const updateAllEnvironments = useUpdateAllEnvironments();
  const { branch, validateDockerfilePath, findDockerfileCaseMatch, getDockerfilesForContext } =
    useRepoTree();

  const {
    handleSubmit,
    formState: { isValid, isSubmitting, errors },
    control,
    setValue,
  } = useForm<z.infer<typeof dockerfileSchema>>({
    resolver: zodResolver(dockerfileSchema),
    mode: "onChange",
    values: { dockerfile: defaultValue },
  });

  const currentDockerfile = useWatch({ control, name: "dockerfile" });

  // An empty path is valid: it means the app builds with Railpack instead.
  const validation = currentDockerfile
    ? validateDockerfilePath(currentDockerfile, dockerContext)
    : "valid";
  const hint = pathHint(
    validation,
    () => findDockerfileCaseMatch(currentDockerfile, dockerContext),
    branch,
  );
  const detectedDockerfiles = getDockerfilesForContext(dockerContext);

  const options = useMemo(
    () => [
      // The empty value is a first-class choice: it means "no Dockerfile
      // configured" and the app is built with Railpack instead. searchValue
      // keeps the option matchable since cmdk can't match an empty string.
      {
        label: <span className="text-gray-11">Automatic (no Dockerfile)</span>,
        selectedLabel: "Automatic (no Dockerfile)",
        value: "",
        searchValue: "automatic (no dockerfile)",
      },
      ...detectedDockerfiles.map((path) => ({
        label: path,
        selectedLabel: path,
        value: path,
        searchValue: path,
      })),
    ],
    [detectedDockerfiles],
  );

  const dirty = currentDockerfile !== defaultValue;
  const saveState = formSaveState({
    isSubmitting,
    isValid,
    isDirty: dirty,
  });

  const onSubmit = async (values: z.infer<typeof dockerfileSchema>) => {
    updateAllEnvironments((draft) => {
      draft.dockerfile = values.dockerfile;
    });
  };

  const inputVariant = pathInputVariant(Boolean(errors.dockerfile), hint);
  const warningMessage = pathHintMessage(hint, "File", (path) =>
    setValue("dockerfile", path, { shouldValidate: true }),
  );

  return (
    <FormSettingCard
      title="Dockerfile"
      onSubmit={handleSubmit(onSubmit)}
      dirty={dirty}
      saveState={saveState}
      autoSave={autoSave}
    >
      <SettingField>
        <FormCombobox
          aria-label="Dockerfile"
          description={warningMessage}
          options={options}
          value={currentDockerfile}
          onSelect={(val) => setValue("dockerfile", val, { shouldValidate: true })}
          creatable
          searchPlaceholder="Search or type a path..."
          emptyMessage={<div className="mt-2">No Dockerfiles detected in repository</div>}
          placeholder={<span className="text-grayA-8">None (auto-detected build)</span>}
          variant={inputVariant}
        />
      </SettingField>
    </FormSettingCard>
  );
};

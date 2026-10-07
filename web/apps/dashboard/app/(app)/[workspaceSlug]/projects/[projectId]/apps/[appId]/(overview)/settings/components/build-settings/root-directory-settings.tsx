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

const dockerContextSegment = /^[A-Za-z0-9._-]+$/;

const rootDirectorySchema = z.object({
  dockerContext: z
    .string()
    .min(1, "Enter a root directory or use '.' for the repository root.")
    .refine(
      (path) =>
        path === "." ||
        (path === path.trim() &&
          !path.startsWith("/") &&
          !path.includes("\\") &&
          path
            .split("/")
            .every(
              (segment) =>
                segment !== "." && segment !== ".." && dockerContextSegment.test(segment),
            )),
      "Enter a relative path like 'api' or 'services/api'. Do not start with '/' or './'.",
    ),
});

export const RootDirectory = () => {
  const { settings, autoSave } = useEnvironmentSettings();
  const { dockerContext: defaultValue } = settings;
  const updateAllEnvironments = useUpdateAllEnvironments();
  const { branch, validatePath, findCaseInsensitiveMatch, rootDirectorySuggestions } =
    useRepoTree();

  const {
    handleSubmit,
    formState: { isValid, isSubmitting, errors },
    control,
    setValue,
  } = useForm<z.infer<typeof rootDirectorySchema>>({
    resolver: zodResolver(rootDirectorySchema),
    mode: "onChange",
    values: { dockerContext: defaultValue },
  });

  const currentDockerContext = useWatch({ control, name: "dockerContext" });

  const validation = validatePath(currentDockerContext, "tree");
  const hint = pathHint(
    validation,
    () => findCaseInsensitiveMatch(currentDockerContext, "tree"),
    branch,
  );
  const options = useMemo(
    () =>
      rootDirectorySuggestions.map(({ path, marker }) => ({
        label: (
          <span className="flex w-full items-center justify-between gap-4">
            <span className="truncate">{path}</span>
            <span className="shrink-0 text-gray-9">{marker}</span>
          </span>
        ),
        selectedLabel: path,
        value: path,
        searchValue: path === "." ? ". repository root" : path,
      })),
    [rootDirectorySuggestions],
  );

  const dirty = currentDockerContext !== defaultValue;
  const saveState = formSaveState({
    isSubmitting,
    isValid,
    isDirty: dirty,
  });

  const onSubmit = async (values: z.infer<typeof rootDirectorySchema>) => {
    updateAllEnvironments((draft) => {
      draft.dockerContext = values.dockerContext;
    });
  };

  const inputVariant = pathInputVariant(Boolean(errors.dockerContext), hint);
  const warningMessage = pathHintMessage(hint, "Directory", (path) =>
    setValue("dockerContext", path, { shouldValidate: true }),
  );

  return (
    <FormSettingCard
      title="Root directory"
      onSubmit={handleSubmit(onSubmit)}
      dirty={dirty}
      saveState={saveState}
      autoSave={autoSave}
    >
      <SettingField>
        <FormCombobox
          aria-label="Root directory"
          description={warningMessage}
          error={errors.dockerContext?.message}
          variant={inputVariant}
          options={options}
          value={currentDockerContext}
          onSelect={(value) => setValue("dockerContext", value, { shouldValidate: true })}
          creatable
          searchPlaceholder="Search or enter a directory..."
          emptyMessage={<div className="mt-2">No app directories detected</div>}
          placeholder={<span className="text-grayA-8">services/api</span>}
        />
      </SettingField>
    </FormSettingCard>
  );
};

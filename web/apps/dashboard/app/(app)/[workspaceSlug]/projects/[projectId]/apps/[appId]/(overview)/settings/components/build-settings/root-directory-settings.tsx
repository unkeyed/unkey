import { zodResolver } from "@hookform/resolvers/zod";
import { IconFolderLinkOutline18 } from "nucleo-ui-outline-18";
import { useMemo } from "react";
import { useForm, useWatch } from "react-hook-form";
import { z } from "zod";
import { FormCombobox } from "@/components/ui/form-combobox";
import { useEnvironmentSettings } from "../../environment-provider";
import { useUpdateAllEnvironments } from "../../hooks/use-update-all-environments";
import { SettingField } from "../shared/form-blocks";
import { FormSettingCard, resolveSaveState } from "../shared/form-setting-card";
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
  const { settings, variant } = useEnvironmentSettings();
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
    defaultValues: { dockerContext: defaultValue },
  });

  const currentDockerContext = useWatch({ control, name: "dockerContext", defaultValue });

  const validation = validatePath(currentDockerContext, "tree");
  const caseMatch =
    validation === "invalid" ? findCaseInsensitiveMatch(currentDockerContext, "tree") : null;
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
      ) : branch ? (
        <span>
          Directory not found on branch <span className="font-medium text-gray-12">{branch}</span>
        </span>
      ) : (
        "Directory not found on this branch"
      )
    ) : undefined;

  return (
    <FormSettingCard
      icon={<IconFolderLinkOutline18 className="text-gray-12" />}
      title="Root directory"
      description="The directory your app lives in. Unkey builds from here. Set it when your app is in a subdirectory (e.g., services/api)."
      displayValue={defaultValue || "."}
      onSubmit={handleSubmit(onSubmit)}
      saveState={saveState}
      autoSave={variant === "onboarding"}
    >
      <SettingField>
        <FormCombobox
          label="Root directory"
          requirement="required"
          description={
            warningMessage ??
            "Select a suggested app directory or enter any repository-relative path. Changes apply on next deploy."
          }
          error={errors.dockerContext?.message}
          variant={inputVariant}
          options={options}
          wrapperClassName="max-w-[calc(var(--setting-w)-1rem)]"
          className="max-w-[calc(var(--setting-w)-1rem)]"
          value={currentDockerContext}
          onSelect={(value) => setValue("dockerContext", value, { shouldValidate: true })}
          creatable
          searchPlaceholder="Search or enter a directory..."
          emptyMessage={<div className="mt-2">No app directories detected</div>}
          placeholder={<span className="text-grayA-8">.</span>}
        />
      </SettingField>
    </FormSettingCard>
  );
};

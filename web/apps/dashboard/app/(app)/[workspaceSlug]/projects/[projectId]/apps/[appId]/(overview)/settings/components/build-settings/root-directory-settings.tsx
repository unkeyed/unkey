import { FormCombobox } from "@/components/ui/form-combobox";
import { zodResolver } from "@hookform/resolvers/zod";
import { FolderLink } from "@unkey/icons";
import { useMemo } from "react";
import { useForm, useWatch } from "react-hook-form";
import { z } from "zod";
import { useEnvironmentSettings } from "../../environment-provider";
import { useUpdateAllEnvironments } from "../../hooks/use-update-all-environments";
import { SettingField } from "../shared/form-blocks";
import { FormSettingCard, resolveSaveState } from "../shared/form-setting-card";
import { useRepoTree } from "./use-repo-tree";

const rootDirectorySchema = z.object({
  dockerContext: z.string(),
});

function normalizeRootDirectory(value: string): string {
  const normalized = value
    .trim()
    .replace(/^(\.\/)+/, "")
    .replace(/^\/+|\/+$/g, "");
  return normalized || ".";
}

function formatRootDirectory(value: string): string {
  const normalized = normalizeRootDirectory(value);
  return normalized === "." ? "Repository root (.)" : normalized;
}

export const RootDirectory = () => {
  const { settings, variant } = useEnvironmentSettings();
  const { dockerContext: defaultValue } = settings;
  const updateAllEnvironments = useUpdateAllEnvironments();
  const { branch, validatePath, findCaseInsensitiveMatch, getDirectories } = useRepoTree();

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
  const detectedDirectories = getDirectories();

  const options = useMemo(
    () => [
      {
        label: <span className="text-gray-11">Repository root (.)</span>,
        selectedLabel: "Repository root (.)",
        value: ".",
        searchValue: "repository root .",
      },
      ...detectedDirectories.map((path) => ({
        label: path,
        selectedLabel: path,
        value: path,
        searchValue: path,
      })),
    ],
    [detectedDirectories],
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
      icon={<FolderLink className="text-gray-12" iconSize="xl-medium" />}
      title="Root directory"
      description="The directory your app lives in. Unkey builds from here. Set it when your app is in a subdirectory (e.g., services/api)."
      displayValue={formatRootDirectory(defaultValue)}
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
            "Unkey detects and builds your app from this directory. For Dockerfile builds it is the build context. Changes apply on next deploy."
          }
          options={options}
          wrapperClassName="max-w-[calc(var(--setting-w)-1rem)]"
          className="max-w-[calc(var(--setting-w)-1rem)]"
          value={currentDockerContext}
          onSelect={(val) =>
            setValue("dockerContext", normalizeRootDirectory(val), { shouldValidate: true })
          }
          creatable
          searchPlaceholder="Search or type a directory..."
          emptyMessage={<div className="mt-2">No directories detected in repository</div>}
          placeholder="."
          error={errors.dockerContext?.message}
          variant={inputVariant}
        />
      </SettingField>
    </FormSettingCard>
  );
};

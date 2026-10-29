import { FormCombobox } from "@/components/ui/form-combobox";
import { SettingsRow, SettingsRowContent, SettingsRowHeader, SettingsRowTitle } from "@unkey/ui";
import { useMemo } from "react";
import type { BuildRecipeForm } from "../hooks/use-build-recipe";
import { pathHint, pathInputVariant } from "./path-hint";
import { pathHintMessage } from "./path-hint-message";
import { useRepoTree } from "./use-repo-tree";

export function RootDirectory({ form }: { form: BuildRecipeForm }) {
  const { branch, validatePath, findCaseInsensitiveMatch, rootDirectorySuggestions } =
    useRepoTree();

  const currentDockerContext = form.watch("dockerContext");
  const setDockerContext = (path: string) =>
    form.setValue("dockerContext", path, { shouldDirty: true, shouldValidate: true });
  const error = form.formState.errors.dockerContext;

  const hint = pathHint(
    validatePath(currentDockerContext, "tree"),
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

  return (
    <SettingsRow>
      <SettingsRowHeader>
        <SettingsRowTitle>Root directory</SettingsRowTitle>
      </SettingsRowHeader>
      <SettingsRowContent>
        <FormCombobox
          aria-label="Root directory"
          className="max-w-(--setting-w)"
          description={pathHintMessage(hint, "Directory", setDockerContext)}
          error={error?.message}
          variant={pathInputVariant(Boolean(error), hint)}
          options={options}
          value={currentDockerContext}
          onSelect={setDockerContext}
          creatable
          searchPlaceholder="Search or enter a directory..."
          emptyMessage={<div className="mt-2">No app directories detected</div>}
          placeholder={<span className="text-grayA-8">services/api</span>}
        />
      </SettingsRowContent>
    </SettingsRow>
  );
}

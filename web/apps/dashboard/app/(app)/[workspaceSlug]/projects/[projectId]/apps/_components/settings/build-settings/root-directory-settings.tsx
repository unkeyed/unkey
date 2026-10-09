import { FormCombobox } from "@/components/ui/form-combobox";
import { SettingsForm, SettingsRow } from "@unkey/ui";
import { useMemo } from "react";
import { z } from "zod";
import { useSettingForm } from "../hooks/use-setting-form";
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

export function RootDirectory() {
  const { form, formProps } = useSettingForm({
    schema: rootDirectorySchema,
    read: (s) => ({ dockerContext: s.dockerContext }),
    write: (draft, values) => {
      draft.dockerContext = values.dockerContext;
    },
  });
  const { branch, validatePath, findCaseInsensitiveMatch, rootDirectorySuggestions } =
    useRepoTree();

  const currentDockerContext = form.watch("dockerContext");
  const setDockerContext = (path: string) =>
    form.setValue("dockerContext", path, { shouldValidate: true });
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
    <SettingsForm {...formProps}>
      <SettingsRow title="Root directory">
        <FormCombobox
          aria-label="Root directory"
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
      </SettingsRow>
    </SettingsForm>
  );
}

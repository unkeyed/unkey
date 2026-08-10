"use client";

import { FormCombobox } from "@/components/ui/form-combobox";
import { cn } from "@/lib/utils";
import { zodResolver } from "@hookform/resolvers/zod";
import { Eye, Plus } from "@unkey/icons";
import { FormInput } from "@unkey/ui";
import { useCallback, useRef } from "react";
import { useFieldArray, useForm, useWatch } from "react-hook-form";
import { z } from "zod";
import { useEnvironmentSettings } from "../../environment-provider";
import { useUpdateAllEnvironments } from "../../hooks/use-update-all-environments";
import { SettingDescription, SettingField } from "../shared/form-blocks";
import { FormSettingCard, resolveSaveState } from "../shared/form-setting-card";
import { RemoveButton } from "../shared/remove-button";
import { useRepoTree } from "./use-repo-tree";

const watchPathsSchema = z.object({
  paths: z.array(
    z.object({
      value: z.string(),
    }),
  ),
});

type WatchPathsForm = z.infer<typeof watchPathsSchema>;

function toFormPaths(paths: string[]): { value: string }[] {
  return paths.map((p) => ({ value: p }));
}

function fromFormPaths(paths: { value: string }[]): string[] {
  return paths.map((p) => p.value).filter(Boolean);
}

function changed<T>(a: T, b: T): boolean {
  return JSON.stringify(a) !== JSON.stringify(b);
}

export const WatchPaths = () => {
  const { settings } = useEnvironmentSettings();
  const defaultPaths = settings.watchPaths ?? [];
  const updateAllEnvironments = useUpdateAllEnvironments();
  const { watchPathSuggestions } = useRepoTree();

  const {
    register,
    handleSubmit,
    formState: { isValid, isSubmitting, errors },
    control,
    reset,
    setValue,
  } = useForm<WatchPathsForm>({
    resolver: zodResolver(watchPathsSchema),
    mode: "onChange",
    defaultValues: { paths: toFormPaths(defaultPaths) },
  });

  const { fields, append, remove } = useFieldArray({
    control,
    name: "paths",
  });

  const inputRefs = useRef<Map<number, HTMLInputElement>>(new Map());
  const setInputRef = useCallback((index: number, el: HTMLInputElement | null) => {
    if (el) {
      inputRefs.current.set(index, el);
    } else {
      inputRefs.current.delete(index);
    }
  }, []);

  const removeAndFocus = useCallback(
    (index: number) => {
      remove(index);
      const focusIndex = index > 0 ? index - 1 : 0;
      requestAnimationFrame(() => {
        inputRefs.current.get(focusIndex)?.focus();
      });
    },
    [remove],
  );

  const appendAndFocus = useCallback(() => {
    append({ value: "" });
    inputRefs.current.get(fields.length)?.focus();
  }, [append, fields.length]);

  const currentPaths = useWatch({ control, name: "paths" });
  const currentValues = fromFormPaths(currentPaths ?? []);
  const hasChanges = changed(defaultPaths, currentValues);
  const rootWatchPath =
    settings.dockerContext && settings.dockerContext !== "."
      ? `${settings.dockerContext}/**`
      : null;
  const options = [...watchPathSuggestions]
    .sort((a, b) => {
      if (a.path === rootWatchPath) {
        return -1;
      }
      if (b.path === rootWatchPath) {
        return 1;
      }
      return a.path.localeCompare(b.path);
    })
    .map(({ path, marker }) => ({
      label: (
        <span className="flex w-full items-center justify-between gap-4">
          <span className="truncate font-mono">{path}</span>
          <span className="shrink-0 text-gray-9">
            {path === rootWatchPath ? "Root directory" : marker}
          </span>
        </span>
      ),
      selectedLabel: path,
      value: path,
      searchValue: path,
      disabled: currentValues.includes(path),
    }));

  const addWatchPath = useCallback(
    (value: string) => {
      if (!value || currentValues.includes(value)) {
        return;
      }

      const emptyIndex = currentPaths?.findIndex((path) => !path.value) ?? -1;
      if (emptyIndex >= 0) {
        setValue(`paths.${emptyIndex}.value`, value, { shouldValidate: true });
        return;
      }
      append({ value });
    },
    [append, currentPaths, currentValues, setValue],
  );

  const saveState = resolveSaveState([
    [isSubmitting, { status: "saving" }],
    [!isValid, { status: "disabled" }],
    [!hasChanges, { status: "disabled", reason: "No changes to save" }],
  ]);

  const onSubmit = async (values: WatchPathsForm) => {
    const watchPaths = fromFormPaths(values.paths);
    updateAllEnvironments((draft) => {
      draft.watchPaths = watchPaths;
    });
    reset({ paths: toFormPaths(watchPaths) });
  };

  const displayValue =
    defaultPaths.length > 0 ? (
      <span className="flex items-center gap-1 truncate">
        <span className="truncate">{defaultPaths[0]}</span>
        {defaultPaths.length > 1 && (
          <span className="shrink-0 text-gray-9">+{defaultPaths.length - 1}</span>
        )}
      </span>
    ) : (
      "All files (no filter)"
    );

  return (
    <FormSettingCard
      icon={<Eye className="text-gray-12" iconSize="xl-medium" />}
      title="Watch paths"
      description="Only trigger deployments when files matching these glob patterns change. Leave empty to deploy on all changes."
      displayValue={displayValue}
      onSubmit={handleSubmit(onSubmit)}
      saveState={saveState}
    >
      <SettingField>
        <span className="text-gray-11 text-[13px] flex items-center">Watch paths</span>
        {fields.map((field, index) => {
          const { ref: rhfRef, ...fieldProps } = register(`paths.${index}.value`);
          return (
            <div key={field.id} className="flex items-start gap-2">
              <FormInput
                className="flex-1 [&_input]:font-mono"
                placeholder="e.g. src/** or services/api/**"
                error={errors.paths?.[index]?.value?.message}
                {...fieldProps}
                ref={(el: HTMLInputElement | null) => {
                  rhfRef(el);
                  setInputRef(index, el);
                }}
                onKeyDown={(e) => {
                  if (e.key === "Enter") {
                    e.preventDefault();
                    appendAndFocus();
                  }
                  if (e.key === "Backspace" && !currentPaths?.[index]?.value) {
                    e.preventDefault();
                    removeAndFocus(index);
                  }
                }}
              />
              <RemoveButton
                onClick={() => removeAndFocus(index)}
                className={cn("shrink-0 transition-opacity duration-150")}
              />
            </div>
          );
        })}
        <FormCombobox
          options={options}
          value=""
          onSelect={addWatchPath}
          creatable
          leftIcon={<Plus iconSize="sm-regular" />}
          searchPlaceholder="Search suggestions or enter a glob..."
          emptyMessage={<div className="mt-2">No suggested watch paths detected</div>}
          placeholder={<span className="text-grayA-8">Add a watch path...</span>}
        />
      </SettingField>
      <SettingDescription>
        Glob patterns (e.g. src/**, **/*.go). Deployments are skipped when no changed files match.
      </SettingDescription>
    </FormSettingCard>
  );
};

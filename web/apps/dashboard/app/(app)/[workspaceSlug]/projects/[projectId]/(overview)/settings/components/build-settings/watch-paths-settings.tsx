"use client";

import { cn } from "@/lib/utils";
import { zodResolver } from "@hookform/resolvers/zod";
import { Eye, Plus } from "@unkey/icons";
import { FormInput } from "@unkey/ui";
import { useCallback, useRef } from "react";
import { useFieldArray, useForm, useWatch } from "react-hook-form";
import { z } from "zod";
import { useEnvironmentSettings } from "../../environment-provider";
import { useUpdateAllEnvironments } from "../../hooks/use-update-all-environments";
import { FormSettingCard, resolveSaveState } from "../shared/form-setting-card";
import { RemoveButton } from "../shared/remove-button";
import { SettingDescription } from "../shared/setting-description";

const watchPathsSchema = z.object({
  paths: z.array(
    z.object({
      value: z.string().min(1, "Pattern cannot be empty"),
    }),
  ),
});

type WatchPathsForm = z.infer<typeof watchPathsSchema>;

function toFormPaths(paths: string[]): { value: string }[] {
  return paths.length > 0 ? paths.map((p) => ({ value: p })) : [{ value: "" }];
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

  const {
    register,
    handleSubmit,
    formState: { isValid, isSubmitting, errors },
    control,
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

  const currentPaths = useWatch({ control, name: "paths" });
  const currentValues = fromFormPaths(currentPaths ?? []);
  const hasChanges = changed(defaultPaths, currentValues);

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
  };

  const displayValue = defaultPaths.length > 0 ? defaultPaths.join(", ") : "All files (no filter)";

  return (
    <FormSettingCard
      icon={<Eye className="text-gray-12" iconSize="xl-medium" />}
      title="Watch paths"
      description="Only trigger deployments when files matching these glob patterns change. Leave empty to deploy on all changes."
      displayValue={displayValue}
      onSubmit={handleSubmit(onSubmit)}
      saveState={saveState}
    >
      <div className="flex flex-col gap-2 w-full">
        {fields.map((field, index) => {
          const isOnly = fields.length === 1;
          const { ref: rhfRef, ...fieldProps } = register(`paths.${index}.value`);
          return (
            <div key={field.id} className="flex items-start gap-2">
              <FormInput
                className="flex-1 [&_input]:h-9 [&_input]:font-mono"
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
                    append({ value: "" });
                  }
                  if (e.key === "Backspace" && !currentPaths?.[index]?.value && !isOnly) {
                    e.preventDefault();
                    removeAndFocus(index);
                  }
                }}
              />
              <RemoveButton
                onClick={() => removeAndFocus(index)}
                className={cn(
                  "shrink-0 transition-opacity duration-150",
                  isOnly ? "opacity-0 pointer-events-none" : "opacity-100",
                )}
              />
            </div>
          );
        })}
        <button
          type="button"
          className="flex items-center gap-1.5 text-gray-9 hover:text-gray-11 text-sm transition-colors w-fit"
          onClick={() => append({ value: "" })}
        >
          <Plus iconSize="sm-regular" />
          Add pattern
        </button>
        <SettingDescription>
          Glob patterns (e.g. src/**, **/*.go). Deployments are skipped when no changed files match.
        </SettingDescription>
      </div>
    </FormSettingCard>
  );
};

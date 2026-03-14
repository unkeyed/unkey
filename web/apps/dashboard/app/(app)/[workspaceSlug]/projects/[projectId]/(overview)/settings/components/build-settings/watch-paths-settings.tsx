"use client";

import { cn } from "@/lib/utils";
import { zodResolver } from "@hookform/resolvers/zod";
import { Eye, Plus } from "@unkey/icons";
import { Button, FormInput } from "@unkey/ui";
import { useFieldArray, useForm, useWatch } from "react-hook-form";
import { z } from "zod";
import { useEnvironmentSettings } from "../../environment-provider";
import { useUpdateAllEnvironments } from "../../hooks/use-update-all-environments";
import { FormSettingCard, resolveSaveState } from "../shared/form-setting-card";
import { SettingDescription } from "../shared/setting-description";
import { RemoveButton } from "../shared/remove-button";

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

  const { fields, prepend, remove } = useFieldArray({
    control,
    name: "paths",
  });

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
          const isFirst = index === 0;
          const isOnly = fields.length === 1;
          return (
            <div key={field.id} className="flex items-start gap-2">
              <FormInput
                className="flex-1 [&_input]:h-9 [&_input]:font-mono"
                placeholder="e.g. src/** or services/api/**"
                error={errors.paths?.[index]?.value?.message}
                {...register(`paths.${index}.value`)}
              />
              <div className="relative w-16 h-9 shrink-0">
                <RemoveButton
                  onClick={() => remove(index)}
                  className={cn(
                    "absolute left-0 transition-opacity duration-150",
                    isOnly ? "opacity-0 pointer-events-none" : "opacity-100",
                  )}
                />
                <Button
                  type="button"
                  variant="ghost"
                  size="sm"
                  className={cn(
                    "absolute left-0 size-9 hover:bg-grayA-3 px-0 justify-center transition-all duration-150 rounded-lg",
                    isOnly ? "translate-x-0" : "translate-x-9",
                    isFirst ? "opacity-100" : "opacity-0 pointer-events-none",
                  )}
                  onClick={() => prepend({ value: "" })}
                >
                  <Plus iconSize="sm-regular" />
                </Button>
              </div>
            </div>
          );
        })}
        <SettingDescription>
          Glob patterns (e.g. src/**, **/*.go). Deployments are skipped when no changed files match.
        </SettingDescription>
      </div>
    </FormSettingCard>
  );
};

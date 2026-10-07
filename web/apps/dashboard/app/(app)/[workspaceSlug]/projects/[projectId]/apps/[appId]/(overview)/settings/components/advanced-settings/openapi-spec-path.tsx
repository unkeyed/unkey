"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { FormInput, formSaveState } from "@unkey/ui";
import { useForm, useWatch } from "react-hook-form";
import { z } from "zod";
import { useEnvironmentSettings } from "../../environment-provider";
import { useUpdateAllEnvironments } from "../../hooks/use-update-all-environments";
import { SettingField } from "../shared/form-blocks";
import { FormSettingCard } from "../shared/form-setting-card";
import { RemoveButton } from "../shared/remove-button";

const openapiSpecPathSchema = z.object({
  openapiSpecPath: z.string().max(512),
});

export const OpenapiSpecPath = () => {
  const { settings, autoSave } = useEnvironmentSettings();
  const { openapiSpecPath } = settings;
  const updateAllEnvironments = useUpdateAllEnvironments();

  const defaultValue = openapiSpecPath ?? "";

  const {
    register,
    handleSubmit,
    reset,
    control,
    formState: { isValid, isSubmitting, errors },
  } = useForm<z.infer<typeof openapiSpecPathSchema>>({
    resolver: zodResolver(openapiSpecPathSchema),
    mode: "onChange",
    values: { openapiSpecPath: defaultValue },
  });

  const current = useWatch({ control, name: "openapiSpecPath" });
  const hasChanges = current !== defaultValue;

  const saveState = formSaveState({
    isSubmitting,
    isValid,
    isDirty: hasChanges,
  });

  const onSubmit = (values: z.infer<typeof openapiSpecPathSchema>) => {
    const trimmed = values.openapiSpecPath.trim();
    updateAllEnvironments((draft) => {
      draft.openapiSpecPath = trimmed === "" ? null : trimmed;
    });
  };

  const handleRemove = () => {
    updateAllEnvironments((draft) => {
      draft.openapiSpecPath = null;
    });
    reset({ openapiSpecPath: "" });
  };

  return (
    <FormSettingCard
      title="OpenAPI spec path"
      onSubmit={handleSubmit(onSubmit)}
      dirty={hasChanges}
      saveState={saveState}
      autoSave={autoSave}
    >
      <SettingField>
        <div className="flex items-start gap-2">
          <FormInput
            data-1p-ignore
            aria-label="OpenAPI spec path"
            placeholder="/openapi.yaml"
            className="flex-1 [&_input]:font-mono"
            error={errors.openapiSpecPath?.message}
            variant={errors.openapiSpecPath ? "error" : "default"}
            {...register("openapiSpecPath")}
          />
          {openapiSpecPath && <RemoveButton onClick={handleRemove} className="shrink-0" />}
        </div>
      </SettingField>
    </FormSettingCard>
  );
};

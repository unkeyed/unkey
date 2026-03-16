"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { BracketsCurly } from "@unkey/icons";
import { FormInput } from "@unkey/ui";
import { useEffect } from "react";
import { useForm, useWatch } from "react-hook-form";
import { z } from "zod";
import { useEnvironmentSettings } from "../../environment-provider";
import { useUpdateAllEnvironments } from "../../hooks/use-update-all-environments";
import { FormSettingCard, resolveSaveState } from "../shared/form-setting-card";
import { RemoveButton } from "../shared/remove-button";

const openapiSpecPathSchema = z.object({
  openapiSpecPath: z.string().max(512),
});

export const OpenapiSpecPath = () => {
  const { settings, variant } = useEnvironmentSettings();
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
    defaultValues: { openapiSpecPath: defaultValue },
  });

  // biome-ignore lint/correctness/useExhaustiveDependencies: we gucci
  useEffect(() => {
    reset({ openapiSpecPath: defaultValue });
  }, [defaultValue, reset]);

  const current = useWatch({ control, name: "openapiSpecPath" });
  const hasChanges = current !== defaultValue;

  const saveState = resolveSaveState([
    [isSubmitting, { status: "saving" }],
    [!isValid, { status: "disabled" }],
    [!hasChanges, { status: "disabled", reason: "No changes to save" }],
  ]);

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
      icon={<BracketsCurly className="text-gray-12" iconSize="xl-medium" />}
      title="OpenAPI Spec Path"
      description="Path to your OpenAPI spec. Leave empty to disable scraping."
      displayValue={
        openapiSpecPath ? (
          <span className="text-gray-12 text-xs">{openapiSpecPath}</span>
        ) : null
      }
      onSubmit={handleSubmit(onSubmit)}
      saveState={saveState}
      autoSave={variant === "onboarding"}
    >
      <div className="flex flex-col gap-2 w-[480px]">
        <span className="text-gray-11 text-[13px]">OpenAPI Spec Path</span>
        <div className="relative flex items-start gap-3">
          <FormInput
            description="Path your deployment serves the OpenAPI spec (e.g. /openapi.yaml). Changes apply on next deploy."
            placeholder="/openapi.yaml"
            className="flex-1 [&_input]:font-mono"
            error={errors.openapiSpecPath?.message}
            variant={errors.openapiSpecPath ? "error" : "default"}
            {...register("openapiSpecPath")}
          />
          {openapiSpecPath && (
            <RemoveButton onClick={handleRemove} className="absolute -right-11 top-0" />
          )}
        </div>
      </div>
    </FormSettingCard>
  );
};

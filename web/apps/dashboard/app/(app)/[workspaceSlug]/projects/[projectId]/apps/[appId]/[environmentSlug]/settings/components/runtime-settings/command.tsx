"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { FormTextarea } from "@unkey/ui";
import { useEffect } from "react";
import { useForm, useWatch } from "react-hook-form";
import { z } from "zod";
import { useEnvironmentSettings } from "../../environment-provider";
import { useUpdateAllEnvironments } from "../../hooks/use-update-all-environments";
import { SettingField } from "../shared/form-blocks";
import { FormSettingCard, resolveSaveState } from "../shared/form-setting-card";

const commandSchema = z.object({
  command: z.string(),
});

type CommandFormValues = z.infer<typeof commandSchema>;

export const Command = () => {
  const { settings, variant } = useEnvironmentSettings();
  const { command } = settings;
  const updateAllEnvironments = useUpdateAllEnvironments();
  const defaultCommand = command.join(" ");

  const {
    register,
    handleSubmit,
    formState: { isValid, isSubmitting, errors },
    control,
    reset,
  } = useForm<CommandFormValues>({
    resolver: zodResolver(commandSchema),
    mode: "onChange",
    defaultValues: { command: defaultCommand },
  });

  useEffect(() => {
    reset({ command: defaultCommand });
  }, [defaultCommand, reset]);

  const currentCommand = useWatch({ control, name: "command" });
  const hasChanges = currentCommand !== defaultCommand;

  const saveState = resolveSaveState([
    [isSubmitting, { status: "saving" }],
    [!isValid, { status: "disabled" }],
    [!hasChanges, { status: "disabled", reason: "No changes to save" }],
  ]);

  const onSubmit = async (values: CommandFormValues) => {
    const trimmed = values.command.trim();
    const command = trimmed === "" ? [] : trimmed.split(/\s+/).filter(Boolean);
    updateAllEnvironments((draft) => {
      draft.command = command;
    });
  };

  return (
    <FormSettingCard
      title="Command"
      description="Overrides the image's startup command. Arguments are split on whitespace. Leave empty to use the image default."
      requirement="optional"
      onSubmit={handleSubmit(onSubmit)}
      saveState={saveState}
      autoSave={variant === "onboarding"}
    >
      <SettingField>
        <FormTextarea
          aria-label="Command"
          placeholder="~ npm start"
          className="[&_textarea]:font-mono"
          variant={errors.command ? "error" : "default"}
          {...register("command")}
        />
      </SettingField>
    </FormSettingCard>
  );
};

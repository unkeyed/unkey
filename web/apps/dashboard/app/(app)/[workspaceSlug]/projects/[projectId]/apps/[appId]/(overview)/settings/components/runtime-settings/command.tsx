"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { FormTextarea, formSaveState } from "@unkey/ui";
import { useForm, useWatch } from "react-hook-form";
import { z } from "zod";
import { useEnvironmentSettings } from "../../environment-provider";
import { useUpdateAllEnvironments } from "../../hooks/use-update-all-environments";
import { SettingField } from "../shared/form-blocks";
import { FormSettingCard } from "../shared/form-setting-card";

const commandSchema = z.object({
  command: z.string(),
});

type CommandFormValues = z.infer<typeof commandSchema>;

export const Command = () => {
  const { settings, autoSave } = useEnvironmentSettings();
  const { command } = settings;
  const updateAllEnvironments = useUpdateAllEnvironments();
  const defaultCommand = command.join(" ");

  const {
    register,
    handleSubmit,
    formState: { isValid, isSubmitting, errors },
    control,
  } = useForm<CommandFormValues>({
    resolver: zodResolver(commandSchema),
    mode: "onChange",
    values: { command: defaultCommand },
  });

  const currentCommand = useWatch({ control, name: "command" });
  const hasChanges = currentCommand !== defaultCommand;

  const saveState = formSaveState({
    isSubmitting,
    isValid,
    isDirty: hasChanges,
  });

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
      description="Unkey splits arguments on whitespace."
      onSubmit={handleSubmit(onSubmit)}
      dirty={hasChanges}
      saveState={saveState}
      autoSave={autoSave}
    >
      <SettingField>
        <FormTextarea
          data-1p-ignore
          aria-label="Command"
          placeholder="Image default"
          className="[&_textarea]:font-mono"
          variant={errors.command ? "error" : "default"}
          {...register("command")}
        />
      </SettingField>
    </FormSettingCard>
  );
};

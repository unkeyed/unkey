"use client";

import { collection } from "@/lib/collections";
import { zodResolver } from "@hookform/resolvers/zod";
import { eq, useLiveQuery } from "@tanstack/react-db";
import { SquareTerminal } from "@unkey/icons";
import { FormTextarea, InfoTooltip } from "@unkey/ui";
import { useEffect } from "react";
import { useForm, useWatch } from "react-hook-form";
import { z } from "zod";
import { useProjectData } from "../../../data-provider";
import { FormSettingCard } from "../shared/form-setting-card";

const commandSchema = z.object({
  command: z.string(),
});

type CommandFormValues = z.infer<typeof commandSchema>;

export const Command = () => {
  const { environments } = useProjectData();
  const environmentId = environments[0]?.id;

  const { data: settings } = useLiveQuery(
    (q) =>
      q
        .from({ s: collection.environmentSettings })
        .where(({ s }) => eq(s.environmentId, environmentId ?? "")),
    [environmentId],
  );

  const rawCommand = settings?.[0]?.command;
  const defaultCommand = (rawCommand ?? []).join(" ");

  return <CommandForm environmentId={environmentId ?? ""} defaultCommand={defaultCommand} />;
};

type CommandFormProps = {
  environmentId: string;
  defaultCommand: string;
};

const CommandForm: React.FC<CommandFormProps> = ({ environmentId, defaultCommand }) => {
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

  const onSubmit = async (values: CommandFormValues) => {
    const trimmed = values.command.trim();
    const command = trimmed === "" ? [] : trimmed.split(/\s+/).filter(Boolean);
    collection.environmentSettings.update(environmentId, (draft) => {
      draft.command = command;
    });
  };

  return (
    <FormSettingCard
      icon={<SquareTerminal className="text-gray-12" iconSize="xl-medium" />}
      title="Command"
      description="The command to start your application. Changes apply on next deploy."
      displayValue={
        defaultCommand ? (
          <InfoTooltip content={defaultCommand} asChild position={{ side: "bottom" }}>
            <span className="font-medium text-gray-12 font-mono text-xs truncate max-w-[100px]">
              {defaultCommand}
            </span>
          </InfoTooltip>
        ) : (
          <span className="text-gray-11 font-normal">Default</span>
        )
      }
      onSubmit={handleSubmit(onSubmit)}
      canSave={isValid && !isSubmitting && hasChanges}
      isSaving={isSubmitting}
    >
      <FormTextarea
        label="Command"
        placeholder="~ npm start"
        className="w-[480px] [&_textarea]:font-mono"
        description="
        Overrides the default container startup command. Arguments are split on whitespace. Leave
        empty to use the image's default command."
        variant={errors.command ? "error" : "default"}
        {...register("command")}
      />
    </FormSettingCard>
  );
};

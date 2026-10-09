"use client";

import {
  FormTextarea,
  SettingsForm,
  SettingsRow,
  SettingsRowContent,
  SettingsRowDescription,
  SettingsRowHeader,
  SettingsRowTitle,
} from "@unkey/ui";
import { z } from "zod";
import { useSettingForm } from "../hooks/use-setting-form";

const commandSchema = z.object({
  command: z.string(),
});

export function Command() {
  const { form, formProps } = useSettingForm({
    schema: commandSchema,
    read: (s) => ({ command: s.command.join(" ") }),
    write: (draft, values) => {
      const trimmed = values.command.trim();
      draft.command = trimmed === "" ? [] : trimmed.split(/\s+/).filter(Boolean);
    },
  });

  return (
    <SettingsForm {...formProps}>
      <SettingsRow>
        <SettingsRowHeader>
          <SettingsRowTitle>Command</SettingsRowTitle>
          <SettingsRowDescription>Unkey splits arguments on whitespace.</SettingsRowDescription>
        </SettingsRowHeader>
        <SettingsRowContent>
          <FormTextarea
            data-1p-ignore
            aria-label="Command"
            placeholder="Image default"
            className="max-w-(--setting-w) [&_textarea]:font-mono"
            variant={form.formState.errors.command ? "error" : "default"}
            {...form.register("command")}
          />
        </SettingsRowContent>
      </SettingsRow>
    </SettingsForm>
  );
}

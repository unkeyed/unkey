"use client";

import { FormTextarea, SettingsForm, SettingsRow } from "@unkey/ui";
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
      <SettingsRow title="Command" description="Unkey splits arguments on whitespace.">
        <FormTextarea
          data-1p-ignore
          aria-label="Command"
          placeholder="Image default"
          className="[&_textarea]:font-mono"
          variant={form.formState.errors.command ? "error" : "default"}
          {...form.register("command")}
        />
      </SettingsRow>
    </SettingsForm>
  );
}

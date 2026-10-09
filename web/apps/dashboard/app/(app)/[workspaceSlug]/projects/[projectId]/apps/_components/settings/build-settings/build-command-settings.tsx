import {
  FormInput,
  SettingsForm,
  SettingsRow,
  SettingsRowContent,
  SettingsRowDescription,
  SettingsRowHeader,
  SettingsRowTitle,
} from "@unkey/ui";
import { z } from "zod";
import { useEnvironmentSettings } from "../environment-provider";
import { useSettingForm } from "../hooks/use-setting-form";

const buildCommandSchema = z.object({
  buildCommand: z.string().max(1000),
});

export function BuildCommand() {
  const dockerfileConfigured = useEnvironmentSettings().settings.dockerfile.trim() !== "";
  const { form, formProps } = useSettingForm({
    schema: buildCommandSchema,
    read: (s) => ({ buildCommand: s.buildCommand }),
    write: (draft, values) => {
      draft.buildCommand = values.buildCommand;
    },
    blockedReason: dockerfileConfigured
      ? "Not used with a Dockerfile build. Remove the Dockerfile to build automatically."
      : undefined,
  });

  return (
    <SettingsForm {...formProps}>
      <SettingsRow>
        <SettingsRowHeader>
          <SettingsRowTitle>Build command</SettingsRowTitle>
          {dockerfileConfigured ? (
            <SettingsRowDescription>Not used with a Dockerfile.</SettingsRowDescription>
          ) : null}
        </SettingsRowHeader>
        <SettingsRowContent>
          <FormInput
            data-1p-ignore
            aria-label="Build command"
            className="max-w-(--setting-w)"
            placeholder="Auto-detected"
            disabled={dockerfileConfigured}
            error={form.formState.errors.buildCommand?.message}
            {...form.register("buildCommand")}
          />
        </SettingsRowContent>
      </SettingsRow>
    </SettingsForm>
  );
}

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
import { useSettingField } from "../hooks/use-setting-form";

export function BuildCommand() {
  const dockerfileConfigured = useEnvironmentSettings().settings.dockerfile.trim() !== "";
  const { form, formProps } = useSettingField("buildCommand", z.string().max(1000), {
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

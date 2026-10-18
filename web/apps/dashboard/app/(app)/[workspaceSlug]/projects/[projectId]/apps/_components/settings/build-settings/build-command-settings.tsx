import {
  FormInput,
  SettingsForm,
  SettingsRow,
  SettingsRowContent,
  SettingsRowDescription,
  SettingsRowHeader,
  SettingsRowTitle,
} from "@unkey/ui";
import type { SettingField } from "../hooks/use-setting-form";

export function BuildCommand({
  field,
  disabled,
}: {
  field: SettingField<"buildCommand">;
  disabled: boolean;
}) {
  const { form, formProps } = field;

  return (
    <SettingsForm {...formProps}>
      <SettingsRow>
        <SettingsRowHeader>
          <SettingsRowTitle>Build command</SettingsRowTitle>
          {disabled ? (
            <SettingsRowDescription>Not used with a Dockerfile.</SettingsRowDescription>
          ) : null}
        </SettingsRowHeader>
        <SettingsRowContent>
          <FormInput
            data-1p-ignore
            aria-label="Build command"
            className="max-w-(--setting-w)"
            placeholder="Auto-detected"
            disabled={disabled}
            error={form.formState.errors.buildCommand?.message}
            {...form.register("buildCommand")}
          />
        </SettingsRowContent>
      </SettingsRow>
    </SettingsForm>
  );
}

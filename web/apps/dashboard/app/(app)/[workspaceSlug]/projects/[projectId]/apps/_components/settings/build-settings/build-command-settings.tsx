import {
  FormInput,
  SettingsForm,
  SettingsRow,
  SettingsRowContent,
  SettingsRowDescription,
  SettingsRowHeader,
  SettingsRowTitle,
} from "@unkey/ui";
import type { useBuildFields } from "../hooks/use-build-fields";

export function BuildCommand({
  field,
}: {
  field: ReturnType<typeof useBuildFields>["buildCommand"];
}) {
  const { form, formProps, dockerfileConfigured } = field;

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

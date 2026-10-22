import {
  FormInput,
  SettingsRow,
  SettingsRowContent,
  SettingsRowDescription,
  SettingsRowHeader,
  SettingsRowTitle,
} from "@unkey/ui";
import type { BuildRecipeForm } from "../hooks/use-build-recipe";

export function BuildCommand({ form }: { form: BuildRecipeForm }) {
  const disabled = form.watch("dockerfile").trim() !== "";

  return (
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
  );
}

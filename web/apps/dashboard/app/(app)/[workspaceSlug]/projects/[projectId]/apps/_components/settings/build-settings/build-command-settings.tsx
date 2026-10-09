import { FormInput, SettingsForm, SettingsRow } from "@unkey/ui";
import { z } from "zod";
import { useEnvironmentSettings } from "../environment-provider";
import { useSettingForm } from "../hooks/use-setting-form";

const buildCommandSchema = z.object({
  // Empty means "let Railpack auto-detect the build command".
  buildCommand: z.string().max(1000),
});

export function BuildCommand() {
  // The build command only applies to automatic (Railpack) builds. A Dockerfile
  // build never invokes Railpack, so the command is ignored.
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
      <SettingsRow
        title="Build command"
        description={dockerfileConfigured ? "Not used with a Dockerfile." : undefined}
      >
        <FormInput
          data-1p-ignore
          aria-label="Build command"
          placeholder="Auto-detected"
          disabled={dockerfileConfigured}
          error={form.formState.errors.buildCommand?.message}
          {...form.register("buildCommand")}
        />
      </SettingsRow>
    </SettingsForm>
  );
}

import {
  FormInput,
  SettingsForm,
  SettingsRow,
  SettingsRowContent,
  SettingsRowHeader,
  SettingsRowTitle,
} from "@unkey/ui";
import { z } from "zod";
import { useSettingForm } from "../hooks/use-setting-form";

const portSchema = z.object({
  port: z.number().int().min(1).max(65535),
});

export function Port() {
  const { form, formProps } = useSettingForm({
    schema: portSchema,
    read: (s) => ({ port: s.port }),
    write: (draft, values) => {
      draft.port = values.port;
    },
  });
  const error = form.formState.errors.port;

  return (
    <SettingsForm {...formProps}>
      <SettingsRow>
        <SettingsRowHeader>
          <SettingsRowTitle>Port</SettingsRowTitle>
        </SettingsRowHeader>
        <SettingsRowContent>
          <FormInput
            data-1p-ignore
            type="number"
            onWheelCapture={(e) => e.currentTarget.blur()}
            aria-label="Port"
            className="max-w-(--setting-w)"
            placeholder="8080"
            min={1}
            max={65535}
            error={error?.message}
            variant={error ? "error" : "default"}
            {...form.register("port", { valueAsNumber: true })}
          />
        </SettingsRowContent>
      </SettingsRow>
    </SettingsForm>
  );
}

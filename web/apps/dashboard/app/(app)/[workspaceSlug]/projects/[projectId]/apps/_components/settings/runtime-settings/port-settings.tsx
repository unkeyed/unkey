import {
  FormInput,
  SettingsForm,
  SettingsRow,
  SettingsRowContent,
  SettingsRowHeader,
  SettingsRowTitle,
} from "@unkey/ui";
import { z } from "zod";
import { useSettingField } from "../hooks/use-setting-form";

export function Port() {
  const { form, formProps } = useSettingField("port", z.number().int().min(1).max(65535));
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

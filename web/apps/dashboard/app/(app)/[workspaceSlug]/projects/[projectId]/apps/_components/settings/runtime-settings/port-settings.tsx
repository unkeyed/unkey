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

const PORT_MESSAGE = "Use a port from 1 to 65535";

export function Port() {
  const { form, formProps } = useSettingField(
    "port",
    z
      .number({ error: PORT_MESSAGE })
      .int(PORT_MESSAGE)
      .min(1, PORT_MESSAGE)
      .max(65535, PORT_MESSAGE),
  );
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

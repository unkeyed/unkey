import {
  FormInput,
  SettingsRow,
  SettingsRowContent,
  SettingsRowHeader,
  SettingsRowTitle,
} from "@unkey/ui";
import type { UseFormReturn } from "react-hook-form";
import type { RuntimeValues } from "./model";

export function PortField({ form }: { form: UseFormReturn<RuntimeValues> }) {
  const error = form.formState.errors.port?.message;
  return (
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
          error={error}
          variant={error ? "error" : "default"}
          {...form.register("port", { valueAsNumber: true })}
        />
      </SettingsRowContent>
    </SettingsRow>
  );
}

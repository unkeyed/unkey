import { EnvironmentShutdownSignal } from "@unkey/api/models/components";
import {
  FormField,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  SettingsRow,
  SettingsRowContent,
  SettingsRowHeader,
  SettingsRowTitle,
} from "@unkey/ui";
import { Controller, type UseFormReturn } from "react-hook-form";
import type { RuntimeValues } from "./model";

const SIGNALS: Record<EnvironmentShutdownSignal, string> = {
  SIGTERM: "Sent to your app when its container stops.",
  SIGINT: "Sent to your app when its container stops.",
  SIGQUIT: "Sent to your app when its container stops.",
  SIGKILL: "Stops the app without allowing cleanup.",
};

export function ShutdownSignalField({ form }: { form: UseFormReturn<RuntimeValues> }) {
  return (
    <SettingsRow>
      <SettingsRowHeader>
        <SettingsRowTitle>Shutdown signal</SettingsRowTitle>
      </SettingsRowHeader>
      <SettingsRowContent>
        <Controller
          control={form.control}
          name="shutdownSignal"
          render={({ field, fieldState }) => (
            <FormField
              className="max-w-(--setting-w)"
              error={fieldState.error?.message}
              description={SIGNALS[field.value]}
            >
              {(control) => (
                <Select
                  value={field.value}
                  onValueChange={field.onChange}
                  disabled={form.formState.isSubmitting}
                >
                  <SelectTrigger
                    id={control.id}
                    aria-label="Shutdown signal"
                    aria-invalid={control.invalid}
                    aria-describedby={control.describedBy}
                    onBlur={field.onBlur}
                    ref={field.ref}
                    className="font-mono"
                  >
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {Object.values(EnvironmentShutdownSignal).map((signal) => (
                      <SelectItem key={signal} value={signal} className="font-mono">
                        {signal}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              )}
            </FormField>
          )}
        />
      </SettingsRowContent>
    </SettingsRow>
  );
}

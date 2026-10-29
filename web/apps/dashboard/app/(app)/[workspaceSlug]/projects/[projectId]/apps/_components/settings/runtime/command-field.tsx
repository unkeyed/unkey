import {
  FormTextarea,
  SettingsRow,
  SettingsRowContent,
  SettingsRowDescription,
  SettingsRowHeader,
  SettingsRowTitle,
} from "@unkey/ui";
import type { UseFormReturn } from "react-hook-form";
import type { RuntimeValues } from "./model";

export function CommandField({ form }: { form: UseFormReturn<RuntimeValues> }) {
  return (
    <SettingsRow>
      <SettingsRowHeader>
        <SettingsRowTitle>Start command</SettingsRowTitle>
        <SettingsRowDescription>Unkey splits arguments on whitespace.</SettingsRowDescription>
      </SettingsRowHeader>
      <SettingsRowContent>
        <FormTextarea
          data-1p-ignore
          aria-label="Start command"
          placeholder="Leave empty to use the image default"
          className="max-w-(--setting-w) [&_textarea]:font-mono"
          {...form.register("command")}
        />
      </SettingsRowContent>
    </SettingsRow>
  );
}

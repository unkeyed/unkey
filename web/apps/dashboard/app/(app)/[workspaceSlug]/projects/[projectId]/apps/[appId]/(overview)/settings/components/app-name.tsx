"use client";

import { createAppRequestSchema, useAppMutations } from "@/lib/deploy-data";
import {
  FormInput,
  SettingsForm,
  SettingsRow,
  SettingsRowContent,
  SettingsRowHeader,
  SettingsRowTitle,
} from "@unkey/ui";
import { useSavableForm } from "../../../../_components/settings/hooks/use-setting-form";

const appNameSchema = createAppRequestSchema.pick({ name: true });

export function AppName({ appId, name }: { appId: string; name: string }) {
  const { updateApp } = useAppMutations();
  const { form, formProps } = useSavableForm({
    schema: appNameSchema,
    values: { name },
    save: (values) => updateApp(appId, { name: values.name }),
  });

  return (
    <SettingsForm {...formProps}>
      <SettingsRow>
        <SettingsRowHeader>
          <SettingsRowTitle>Name</SettingsRowTitle>
        </SettingsRowHeader>
        <SettingsRowContent>
          <FormInput
            data-1p-ignore
            aria-label="App name"
            className="max-w-(--setting-w)"
            error={form.formState.errors.name?.message}
            {...form.register("name", { setValueAs: (value: string) => value.trim() })}
          />
        </SettingsRowContent>
      </SettingsRow>
    </SettingsForm>
  );
}

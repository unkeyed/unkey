"use client";

import { collection } from "@/lib/collections";
import { createAppRequestSchema } from "@/lib/collections/deploy/apps";
import { FormInput, SettingsForm, SettingsRow } from "@unkey/ui";
import { useSavableForm } from "../../../../_components/settings/hooks/use-setting-form";

const appNameSchema = createAppRequestSchema.pick({ name: true });

export function AppName({ appId, name }: { appId: string; name: string }) {
  const { form, formProps } = useSavableForm({
    schema: appNameSchema,
    values: { name },
    isEqual: (current, saved) => current.name.trim() === saved.name,
    save: (values) =>
      collection.apps.update(appId, (draft) => {
        draft.name = values.name;
      }).isPersisted.promise,
  });

  return (
    <SettingsForm {...formProps}>
      <SettingsRow title="Name">
        <FormInput
          data-1p-ignore
          aria-label="App name"
          error={form.formState.errors.name?.message}
          {...form.register("name")}
        />
      </SettingsRow>
    </SettingsForm>
  );
}

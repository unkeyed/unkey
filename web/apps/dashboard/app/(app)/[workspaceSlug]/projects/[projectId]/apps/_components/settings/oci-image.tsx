"use client";

import { collection } from "@/lib/collections";
import { ociImageReferenceSchema } from "@/lib/collections/deploy/apps";
import {
  FormInput,
  SettingsForm,
  SettingsRow,
  SettingsRowContent,
  SettingsRowHeader,
  SettingsRowTitle,
} from "@unkey/ui";
import { z } from "zod";
import { useSavableForm } from "./hooks/use-setting-form";

const ociImageFormSchema = z.object({
  imageReference: ociImageReferenceSchema,
});

export function OCIImage({ appId, imageReference }: { appId: string; imageReference: string }) {
  const { form, formProps } = useSavableForm({
    schema: ociImageFormSchema,
    values: { imageReference },
    save: (values) =>
      collection.apps.update(appId, (draft) => {
        draft.imageReference = values.imageReference;
      }).isPersisted.promise,
  });

  return (
    <SettingsForm {...formProps}>
      <SettingsRow>
        <SettingsRowHeader>
          <SettingsRowTitle>Image</SettingsRowTitle>
        </SettingsRowHeader>
        <SettingsRowContent>
          <FormInput
            data-1p-ignore
            aria-label="Image reference"
            className="max-w-(--setting-w)"
            placeholder="ghcr.io/acme/app:v1.2.3"
            error={form.formState.errors.imageReference?.message}
            {...form.register("imageReference")}
          />
        </SettingsRowContent>
      </SettingsRow>
    </SettingsForm>
  );
}

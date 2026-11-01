"use client";

import { ociImageReferenceSchema, useAppMutations } from "@/lib/deploy-data";
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
  const { updateApp } = useAppMutations();
  const { form, formProps } = useSavableForm({
    schema: ociImageFormSchema,
    values: { imageReference },
    save: (values) => updateApp(appId, { imageReference: values.imageReference }),
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
            placeholder="ghcr.io/unkeyed/deploy-demo:latest"
            error={form.formState.errors.imageReference?.message}
            {...form.register("imageReference")}
          />
        </SettingsRowContent>
      </SettingsRow>
    </SettingsForm>
  );
}

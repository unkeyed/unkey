"use client";

import {
  FormInput,
  SettingsForm,
  SettingsRow,
  SettingsRowContent,
  SettingsRowHeader,
  SettingsRowTitle,
} from "@unkey/ui";
import { z } from "zod";
import { RemoveButton } from "../../remove-button";
import { useSettingForm } from "../hooks/use-setting-form";

const openapiSpecPathSchema = z.object({
  openapiSpecPath: z.string().max(512),
});

export function OpenapiSpecPath() {
  const { settings, update, form, formProps } = useSettingForm({
    schema: openapiSpecPathSchema,
    read: (s) => ({ openapiSpecPath: s.openapiSpecPath ?? "" }),
    write: (draft, values) => {
      const trimmed = values.openapiSpecPath.trim();
      draft.openapiSpecPath = trimmed === "" ? null : trimmed;
    },
  });
  const error = form.formState.errors.openapiSpecPath;

  const handleRemove = () => {
    update((draft) => {
      draft.openapiSpecPath = null;
    });
  };

  return (
    <SettingsForm {...formProps}>
      <SettingsRow>
        <SettingsRowHeader>
          <SettingsRowTitle>OpenAPI spec path</SettingsRowTitle>
        </SettingsRowHeader>
        <SettingsRowContent>
          <div className="flex max-w-(--setting-w) items-start gap-2">
            <FormInput
              data-1p-ignore
              aria-label="OpenAPI spec path"
              placeholder="/openapi.yaml"
              className="flex-1 [&_input]:font-mono"
              error={error?.message}
              variant={error ? "error" : "default"}
              {...form.register("openapiSpecPath")}
            />
            {settings.openapiSpecPath ? (
              <RemoveButton
                label="Remove OpenAPI spec path"
                onClick={handleRemove}
                className="shrink-0"
              />
            ) : null}
          </div>
        </SettingsRowContent>
      </SettingsRow>
    </SettingsForm>
  );
}

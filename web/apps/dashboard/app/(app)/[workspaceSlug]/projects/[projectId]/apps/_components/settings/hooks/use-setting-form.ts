"use client";

import type { EnvironmentSettings } from "@/lib/collections/deploy/environment-settings";
import { sameJson } from "@/lib/utils/same-json";
import { zodResolver } from "@hookform/resolvers/zod";
import { formSaveState } from "@unkey/ui";
import type { BaseSyntheticEvent } from "react";
import { type FieldValues, useForm } from "react-hook-form";
import type { z } from "zod";
import { useEnvironmentSettings } from "../environment-provider";

export function useSavableForm<TValues extends FieldValues>({
  schema,
  values,
  save,
  isEqual = sameJson,
  blockedReason,
}: {
  schema: z.ZodType<TValues, TValues>;
  values: TValues;
  save: (values: TValues) => Promise<unknown>;
  isEqual?: (current: TValues, saved: TValues) => boolean;
  blockedReason?: string;
}) {
  const form = useForm<TValues>({ resolver: zodResolver(schema), mode: "onChange", values });
  const { isSubmitting, isValid } = form.formState;
  const dirty = !isEqual(form.watch(), values);

  const submit = async (event?: BaseSyntheticEvent): Promise<boolean> => {
    let saved = false;
    await form.handleSubmit(async (submitted) => {
      saved = await save(submitted).then(
        () => true,
        () => false,
      );
    })(event);
    return saved;
  };

  return {
    form,
    submit,
    formProps: {
      dirty,
      saveState: formSaveState({ isSubmitting, isValid, isDirty: dirty, blockedReason }),
      onSubmit: submit,
    },
  };
}

export function useSettingForm<TValues extends FieldValues>({
  read,
  write,
  ...options
}: {
  schema: z.ZodType<TValues, TValues>;
  read: (settings: EnvironmentSettings) => TValues;
  write: (draft: EnvironmentSettings, values: TValues) => void;
  isEqual?: (current: TValues, saved: TValues) => boolean;
  blockedReason?: string;
}) {
  const context = useEnvironmentSettings();
  const form = useSavableForm({
    ...options,
    values: read(context.settings),
    save: (submitted) => context.update((draft) => write(draft, submitted)).isPersisted.promise,
  });

  return { ...context, ...form };
}

"use client";

import type { EnvironmentSettings } from "@/lib/collections/deploy/environment-settings";
import { sameJson } from "@/lib/utils/same-json";
import { zodResolver } from "@hookform/resolvers/zod";
import { formSaveState } from "@unkey/ui";
import { type BaseSyntheticEvent, useRef } from "react";
import { type FieldValues, useForm } from "react-hook-form";
import type { z } from "zod";
import { useEnvironmentSettings } from "../environment-provider";

export function useSavableForm<TValues extends FieldValues, TResult = unknown>({
  schema,
  values,
  save,
  isEqual = sameJson,
  blockedReason,
}: {
  schema: z.ZodType<TValues, TValues>;
  values: TValues;
  save: (values: TValues) => Promise<TResult>;
  isEqual?: (current: TValues, saved: TValues) => boolean;
  blockedReason?: string;
}) {
  // A failed save rolls the row back, and `values` would reset the form over the edits.
  const heldValues = useRef<TValues | null>(null);
  const form = useForm<TValues>({
    resolver: zodResolver(schema),
    mode: "onChange",
    values: heldValues.current ?? values,
  });
  const { isSubmitting, isValid } = form.formState;
  const dirty = !isEqual(form.watch(), values);

  const submit = async (event?: BaseSyntheticEvent): Promise<TResult | undefined> => {
    let result: TResult | undefined;
    await form.handleSubmit(async (submitted) => {
      heldValues.current = values;
      try {
        result = await save(submitted).catch(() => undefined);
      } finally {
        heldValues.current = null;
      }
    })(event);
    return result;
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
    save: (submitted: TValues) =>
      context
        .update((draft) => write(draft, submitted))
        .isPersisted.promise.then(
          () => true,
          () => false,
        ),
  });

  return { ...context, ...form };
}

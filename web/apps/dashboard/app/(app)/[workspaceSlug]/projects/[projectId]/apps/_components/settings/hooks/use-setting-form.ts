"use client";

import type { EnvironmentSettings } from "@/lib/collections/deploy/environment-settings";
import { sameJson } from "@/lib/utils/same-json";
import { zodResolver } from "@hookform/resolvers/zod";
import { formSaveState } from "@unkey/ui";
import type { BaseSyntheticEvent } from "react";
import { type FieldValues, useForm } from "react-hook-form";
import { z } from "zod";
import { useEnvironmentSettings } from "../environment-provider";

export function useSavableForm<TValues extends FieldValues>({
  schema,
  values,
  save,
  isNoop,
  blockedReason,
}: {
  schema: z.ZodType<TValues, TValues>;
  values: TValues;
  save: (values: TValues) => Promise<unknown>;
  isNoop?: (submitted: TValues) => boolean;
  blockedReason?: string;
}) {
  const form = useForm<TValues>({ resolver: zodResolver(schema), mode: "onChange", values });
  const { isSubmitting, isValid, isDirty } = form.formState;

  const submit = async (event?: BaseSyntheticEvent): Promise<boolean> => {
    let saved = false;
    let noop = false;
    await form.handleSubmit(async (submitted) => {
      noop = isNoop?.(submitted) ?? false;
      saved = await save(submitted).then(
        () => true,
        () => false,
      );
    })(event);
    // A save that normalizes to the stored value leaves `values` unchanged, so RHF never resets on its own.
    if (saved && noop) {
      form.reset(values);
    }
    return saved;
  };

  return {
    form,
    submit,
    formProps: {
      dirty: isDirty,
      saveState: formSaveState({ isSubmitting, isValid, isDirty, blockedReason }),
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
  write: (draft: EnvironmentSettings, values: TValues, saved: TValues) => void;
  blockedReason?: string;
}) {
  const context = useEnvironmentSettings();
  const saved = read(context.settings);
  const form = useSavableForm({
    ...options,
    values: saved,
    save: (submitted) =>
      context.update((draft) => write(draft, submitted, saved)).isPersisted.promise,
    isNoop: (submitted) => {
      const draft = structuredClone(context.settings);
      write(draft, submitted, saved);
      return sameJson(read(draft), saved);
    },
  });

  return { ...context, ...form };
}

type FieldValuesOf<K extends keyof EnvironmentSettings> = Pick<EnvironmentSettings, K>;

export function useSettingField<K extends keyof EnvironmentSettings>(
  key: K,
  field: z.ZodType<EnvironmentSettings[K], EnvironmentSettings[K]>,
  options: { blockedReason?: string } = {},
) {
  return useSettingForm<FieldValuesOf<K>>({
    // A computed key widens to a string index, so the one-key shape is asserted once here.
    schema: z.object({ [key]: field }) as unknown as z.ZodType<FieldValuesOf<K>, FieldValuesOf<K>>,
    read: (settings) => ({ [key]: settings[key] }) as FieldValuesOf<K>,
    write: (draft, values) => {
      draft[key] = values[key];
    },
    ...options,
  });
}

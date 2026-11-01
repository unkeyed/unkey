"use client";

import type { EnvironmentSettings } from "@/lib/collections/deploy/environment-settings";
import { zodResolver } from "@hookform/resolvers/zod";
import { formSaveState } from "@unkey/ui";
import { type BaseSyntheticEvent, useEffect, useRef } from "react";
import { type FieldValues, useForm } from "react-hook-form";
import { z } from "zod";
import { useEnvironmentSettings } from "../environment-provider";

export function useSavableForm<TValues extends FieldValues>({
  schema,
  values,
  save,
}: {
  schema: z.ZodType<TValues, TValues>;
  values: TValues;
  save: (values: TValues) => Promise<unknown>;
}) {
  const form = useForm<TValues>({ resolver: zodResolver(schema), mode: "onChange", values });
  const { isSubmitting, isValid, isDirty } = form.formState;
  const stored = useRef(values);
  useEffect(() => {
    stored.current = values;
  });

  const valuesKey = JSON.stringify(values);
  // biome-ignore lint/correctness/useExhaustiveDependencies: valuesKey is the trigger; stored values can break a rule, so show it before the first edit
  useEffect(() => {
    void form.trigger();
  }, [form, valuesKey]);

  const submit = async (event?: BaseSyntheticEvent): Promise<boolean> => {
    let saved = false;
    await form.handleSubmit(async (submitted) => {
      saved = await save(submitted).then(
        () => true,
        () => false,
      );
    })(event);
    // A save that stores nothing new leaves `values` unchanged, so RHF never resets on its own.
    if (saved) {
      form.reset(stored.current);
    }
    return saved;
  };

  return {
    form,
    submit,
    formProps: {
      dirty: isDirty,
      saveState: formSaveState({ isSubmitting, isValid, isDirty }),
      onSubmit: submit,
    },
  };
}

export function useSettingForm<TValues extends FieldValues>({
  schema,
  read,
  write,
}: {
  schema: z.ZodType<TValues, TValues>;
  read: (settings: EnvironmentSettings) => TValues;
  write: (draft: EnvironmentSettings, values: TValues, saved: TValues) => void;
}) {
  const context = useEnvironmentSettings();
  const saved = read(context.settings);
  const form = useSavableForm({
    schema,
    values: saved,
    save: (submitted) => context.update((draft) => write(draft, submitted, saved)),
  });

  return { ...context, ...form };
}

type FieldValuesOf<K extends keyof EnvironmentSettings> = Pick<EnvironmentSettings, K>;

export function useSettingField<K extends keyof EnvironmentSettings>(
  key: K,
  field: z.ZodType<EnvironmentSettings[K], EnvironmentSettings[K]>,
) {
  return useSettingForm<FieldValuesOf<K>>({
    // A computed key widens to a string index, so the one-key shape is asserted once here.
    schema: z.object({ [key]: field }) as unknown as z.ZodType<FieldValuesOf<K>, FieldValuesOf<K>>,
    read: (settings) => ({ [key]: settings[key] }) as FieldValuesOf<K>,
    write: (draft, values) => {
      draft[key] = values[key];
    },
  });
}

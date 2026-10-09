"use client";

import type { EnvironmentSettings } from "@/lib/collections/deploy/environment-settings";
import { sameJson } from "@/lib/utils/same-json";
import { zodResolver } from "@hookform/resolvers/zod";
import { formSaveState } from "@unkey/ui";
import { type BaseSyntheticEvent, useEffect, useRef } from "react";
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

  return {
    form,
    formProps: {
      dirty,
      saveState: formSaveState({ isSubmitting, isValid, isDirty: dirty, blockedReason }),
      // The collection reports a failed save, so the form only needs to stay put.
      onSubmit: form.handleSubmit((submitted) => save(submitted).catch(() => undefined)),
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
  const values = read(context.settings);
  const isEqual = options.isEqual ?? sameJson;
  const rejected = useRef<TValues | null>(null);

  const persist = (submitted: TValues): Promise<boolean> =>
    context
      .update((draft) => write(draft, submitted))
      .isPersisted.promise.then(
        () => true,
        // The collection reports the failure and rolls the row back.
        () => {
          rejected.current = submitted;
          restoreEdits();
          return false;
        },
      );
  const form = useSavableForm({ ...options, values, save: persist });

  // The rollback resets the form to the saved row, which can land before or
  // after the rejection, so both paths put the user's edits back.
  const restoreEdits = () => {
    const submitted = rejected.current;
    if (submitted && !isEqual(form.form.getValues(), submitted)) {
      rejected.current = null;
      form.form.reset(submitted, { keepDefaultValues: true });
    }
  };
  // biome-ignore lint/correctness/useExhaustiveDependencies: runs once the rolled-back row has rendered
  useEffect(restoreEdits, [context.settings]);

  const submit = async (event?: BaseSyntheticEvent): Promise<boolean> => {
    let saved = false;
    await form.form.handleSubmit(async (submitted) => {
      saved = await persist(submitted);
    })(event);
    return saved;
  };

  return { ...context, ...form, submit };
}

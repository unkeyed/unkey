"use client";

import type { z } from "zod";
import type { ComputeInput } from "../compute/compute-fields";
import { useSettingForm } from "../hooks/use-setting-form";
import { type RuntimeValues, readRuntime, runtimeSchema, writeRuntime } from "./model";

const CHANGE = { shouldDirty: true, shouldValidate: true } as const;

export function useRuntimeForm(schema: z.ZodType<RuntimeValues, RuntimeValues> = runtimeSchema) {
  const { form, formProps } = useSettingForm({
    schema,
    read: readRuntime,
    write: writeRuntime,
  });
  const compute: ComputeInput = {
    values: form.watch("compute"),
    change: (key, value) =>
      form.setValue("compute", { ...form.getValues("compute"), [key]: value }, CHANGE),
    busy: form.formState.isSubmitting,
  };
  return { form, formProps, compute };
}

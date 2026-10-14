"use client";

import { resolveLimits } from "@/lib/compute/sizing";
import { trpc } from "@/lib/trpc/client";
import { useWorkspace } from "@/providers/workspace-provider";
import type { FieldPath, FieldPathValue } from "react-hook-form";
import { useSettingForm } from "../hooks/use-setting-form";
import { type ComputeValues, computeSchema, readCompute, writeCompute } from "./draft";
import { availableFrom } from "./status";

export type SetCompute = <K extends FieldPath<ComputeValues>>(
  key: K,
  value: FieldPathValue<ComputeValues, K>,
) => void;

export type ComputeForm = ReturnType<typeof useCompute>;

export function useCompute() {
  const { limits } = useWorkspace();
  const regionsQuery = trpc.deploy.environmentSettings.getAvailableRegions.useQuery();
  const { form, formProps, submit } = useSettingForm({
    schema: computeSchema,
    read: readCompute,
    write: writeCompute,
  });

  const change: SetCompute = (key, value) => {
    form.setValue(key, value, { shouldDirty: true, shouldValidate: true });
  };

  return {
    form,
    formProps,
    submit,
    values: form.watch(),
    busy: form.formState.isSubmitting,
    change,
    limits: resolveLimits(limits),
    available: availableFrom(regionsQuery),
  };
}

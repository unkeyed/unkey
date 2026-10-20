"use client";

import { resolveLimits } from "@/lib/compute/sizing";
import { useWorkspace } from "@/providers/workspace-provider";
import type { ComputeInput } from "../compute/compute-fields";
import { useSettingForm } from "../hooks/use-setting-form";
import {
  commandSectionSchema,
  computeSectionSchema,
  readRuntime,
  runtimeSchema,
  withReplicaLimit,
  writeRuntime,
} from "./model";

const CHANGE = { shouldDirty: true, shouldValidate: true } as const;

const SHOWN = {
  all: { schema: runtimeSchema, checksCompute: true },
  compute: { schema: computeSectionSchema, checksCompute: true },
  command: { schema: commandSectionSchema, checksCompute: false },
} as const;

export function useRuntimeForm(shown: keyof typeof SHOWN = "all") {
  const { limits } = useWorkspace();
  const { schema, checksCompute } = SHOWN[shown];
  const { form, formProps } = useSettingForm({
    schema: checksCompute ? withReplicaLimit(schema, resolveLimits(limits).replicas) : schema,
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

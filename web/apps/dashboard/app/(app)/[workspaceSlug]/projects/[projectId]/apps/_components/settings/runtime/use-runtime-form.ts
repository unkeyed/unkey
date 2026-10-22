"use client";

import { resolveLimits } from "@/lib/compute/sizing";
import { useWorkspace } from "@/providers/workspace-provider";
import { useEffect } from "react";
import { useSettingForm } from "../hooks/use-setting-form";
import { type RuntimeSection, readRuntime, runtimeFormSchema, writeRuntime } from "./model";

export function useRuntimeForm(shown: readonly RuntimeSection[] | "all" = "all") {
  const { limits } = useWorkspace();
  const replicaLimit = resolveLimits(limits).replicas;
  const runtime = useSettingForm({
    schema: runtimeFormSchema(shown, replicaLimit),
    read: readRuntime,
    write: writeRuntime,
  });
  const { form } = runtime;
  // biome-ignore lint/correctness/useExhaustiveDependencies: the plan limit loads after the form, and the limit check must rerun when it arrives
  useEffect(() => {
    void form.trigger();
  }, [form, replicaLimit]);
  return runtime;
}

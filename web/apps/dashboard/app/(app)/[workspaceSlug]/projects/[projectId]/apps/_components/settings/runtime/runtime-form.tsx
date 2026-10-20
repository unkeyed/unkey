"use client";

import { SettingsForm } from "@unkey/ui";
import { ComputeFields } from "../compute/compute-fields";
import { CommandField } from "./command-field";
import { HealthcheckFields } from "./healthcheck-fields";
import { commandSectionSchema, computeSectionSchema } from "./model";
import { PortField } from "./port-field";
import { useRuntimeForm } from "./use-runtime-form";

export function RuntimeForm() {
  const { form, formProps, compute } = useRuntimeForm();
  return (
    <SettingsForm {...formProps}>
      <ComputeFields compute={compute} />
      <PortField form={form} />
      <CommandField form={form} />
      <HealthcheckFields form={form} />
    </SettingsForm>
  );
}

export function ComputeForm({ hideLockedSizes = false }: { hideLockedSizes?: boolean }) {
  const { formProps, compute } = useRuntimeForm(computeSectionSchema);
  return (
    <SettingsForm {...formProps}>
      <ComputeFields compute={compute} hideLockedSizes={hideLockedSizes} />
    </SettingsForm>
  );
}

export function CommandForm() {
  const { form, formProps } = useRuntimeForm(commandSectionSchema);
  return (
    <SettingsForm {...formProps}>
      <CommandField form={form} />
    </SettingsForm>
  );
}

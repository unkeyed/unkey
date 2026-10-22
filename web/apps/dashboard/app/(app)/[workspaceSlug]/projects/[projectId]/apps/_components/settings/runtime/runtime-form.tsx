"use client";

import { SettingsForm } from "@unkey/ui";
import { ComputeFields } from "../compute/compute-fields";
import { CommandField } from "./command-field";
import { HealthcheckFields } from "./healthcheck-fields";
import { PortField } from "./port-field";
import { useRuntimeForm } from "./use-runtime-form";

export function RuntimeForm() {
  const { form, formProps } = useRuntimeForm();
  return (
    <SettingsForm {...formProps}>
      <ComputeFields form={form} />
      <PortField form={form} />
      <CommandField form={form} />
      <HealthcheckFields form={form} />
    </SettingsForm>
  );
}

export function ComputeForm({ hideLockedSizes = false }: { hideLockedSizes?: boolean }) {
  const { form, formProps } = useRuntimeForm(["compute"]);
  return (
    <SettingsForm {...formProps}>
      <ComputeFields form={form} hideLockedSizes={hideLockedSizes} />
    </SettingsForm>
  );
}

export function CommandForm() {
  const { form, formProps } = useRuntimeForm(["command"]);
  return (
    <SettingsForm {...formProps}>
      <CommandField form={form} />
    </SettingsForm>
  );
}

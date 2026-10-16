"use client";

import { SettingsForm } from "@unkey/ui";
import { ComputeFields } from "./compute-fields";
import { useCompute } from "./use-compute";

export function ComputeForm() {
  const compute = useCompute();
  return (
    <SettingsForm {...compute.formProps}>
      <ComputeFields compute={compute} />
    </SettingsForm>
  );
}

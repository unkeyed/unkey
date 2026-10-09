"use client";

import { onStorageTable, storagePresets, unitFields } from "@/lib/compute/sizing";
import type { ComputeLimits } from "@/lib/compute/sizing";
import { useState } from "react";
import type { ComputeValues } from "./draft";
import type { SetCompute } from "./use-compute";
import { CUSTOM_VALUE } from "./use-size-choice";

export function useStorageChoice({
  values,
  limits,
  onChange,
}: { values: ComputeValues; limits: ComputeLimits; onChange: SetCompute }) {
  const [pickedCustom, setPickedCustom] = useState(() => !onStorageTable(values.storageMib));
  const custom = pickedCustom || !onStorageTable(values.storageMib);
  const pick = (next: string) => {
    setPickedCustom(next === CUSTOM_VALUE);
    const storageMib = Number(next);
    if (next !== CUSTOM_VALUE && Number.isFinite(storageMib)) {
      onChange("storageMib", storageMib);
    }
  };
  return {
    custom,
    selected: custom ? CUSTOM_VALUE : String(values.storageMib),
    options: storagePresets(limits),
    field: unitFields(limits).storage,
    customValue: values.storageMib > 0 ? values.storageMib : null,
    pick,
  };
}

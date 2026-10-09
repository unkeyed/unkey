"use client";

import { CUSTOM_SIZE_LABEL, PRESETS, presetFor, sizeChoice } from "@/lib/compute/sizing";
import { useState } from "react";
import type { ComputeValues } from "./draft";
import type { SetCompute } from "./use-compute";

export const CUSTOM_VALUE = "custom";

export function useSizeChoice({
  values,
  onChange,
}: { values: ComputeValues; onChange: SetCompute }) {
  const [custom, setCustom] = useState(
    () => presetFor(values.cpuMillicores, values.memoryMib) === undefined,
  );
  const choice = sizeChoice(values, custom);
  const pick = (value: string) => {
    const preset = PRESETS.find((p) => p.id === value);
    setCustom(preset === undefined);
    if (preset) {
      onChange("cpuMillicores", preset.cpuMillicores);
      onChange("memoryMib", preset.memoryMib);
    }
  };
  const preset = choice.type === "preset" ? choice.preset : undefined;
  return {
    choice,
    selected: preset?.id ?? CUSTOM_VALUE,
    label: preset?.label ?? CUSTOM_SIZE_LABEL,
    pick,
  };
}

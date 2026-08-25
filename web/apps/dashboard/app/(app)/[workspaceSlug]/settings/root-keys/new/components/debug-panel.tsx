"use client";

import { Checkbox } from "@unkey/ui";
import { useId } from "react";

type DebugPanelProps = {
  debug: boolean;
  onDebugChange: (debug: boolean) => void;
};

export function DebugPanel({ debug, onDebugChange }: DebugPanelProps) {
  const id = useId();

  return (
    <div className="fixed bottom-4 right-4 z-50 flex items-center gap-2 rounded-lg border border-grayA-4 bg-white p-3 shadow-md dark:bg-black">
      <Checkbox
        id={id}
        size="md"
        checked={debug}
        onCheckedChange={(next) => onDebugChange(next === true)}
      />
      <label htmlFor={id} className="cursor-pointer select-none text-xs text-gray-11">
        Debug
      </label>
    </div>
  );
}

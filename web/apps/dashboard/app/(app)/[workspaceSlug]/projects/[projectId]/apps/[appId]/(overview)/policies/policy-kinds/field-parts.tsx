"use client";

import { IconPlusOutline18, IconTrashOutline18 } from "@unkey/icons";
import { Button, FormLabel } from "@unkey/ui";
import type { ReactNode } from "react";

export function FieldSection({
  label,
  htmlFor,
  tooltip,
  onAdd,
  children,
}: {
  label: string;
  htmlFor: string;
  tooltip?: string;
  onAdd: (() => void) | null;
  children: ReactNode;
}) {
  return (
    <fieldset className="flex flex-col gap-2 border-0 m-0 p-0">
      <div className="flex items-center justify-between">
        <FormLabel label={label} htmlFor={htmlFor} tooltipContent={tooltip} />
        {onAdd ? (
          <Button type="button" variant="outline" size="md" className="font-medium" onClick={onAdd}>
            <IconPlusOutline18 />
            Add
          </Button>
        ) : null}
      </div>
      {children}
    </fieldset>
  );
}

export function RemoveRowButton({
  label,
  disabled,
  onClick,
}: {
  label: string;
  disabled?: boolean;
  onClick: () => void;
}) {
  return (
    <Button
      type="button"
      variant="ghost"
      size="sm"
      aria-label={label}
      className="size-9 shrink-0 px-0 justify-center text-gray-11 hover:text-gray-12 hover:bg-grayA-3 rounded-lg"
      disabled={disabled}
      onClick={onClick}
    >
      <IconTrashOutline18 />
    </Button>
  );
}

export function parseOptionalInt(raw: string): number | undefined {
  if (raw === "") {
    return undefined;
  }
  const n = Number.parseInt(raw, 10);
  return Number.isNaN(n) ? undefined : n;
}

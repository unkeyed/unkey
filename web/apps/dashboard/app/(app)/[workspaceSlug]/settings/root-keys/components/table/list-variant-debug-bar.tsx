"use client";

// TODO: temporary review affordance for comparing the two list treatments.
// Delete this file and its usage once we pick one.
import { Button } from "@unkey/ui";

export type ListVariant = "table" | "resource-list";

const VARIANTS: { value: ListVariant; label: string }[] = [
  { value: "table", label: "DataTable" },
  { value: "resource-list", label: "ResourceList" },
];

type ListVariantDebugBarProps = {
  variant: ListVariant;
  onChange: (variant: ListVariant) => void;
};

export function ListVariantDebugBar({ variant, onChange }: ListVariantDebugBarProps) {
  return (
    <div className="sticky bottom-4 z-30 mx-4 mt-4 flex items-center gap-2 rounded-lg border border-dashed border-grayA-5 bg-white px-3 py-2 shadow-sm dark:bg-black">
      <span className="text-xs font-medium text-gray-9">Debug · list treatment</span>
      <div className="ml-auto flex items-center gap-1">
        {VARIANTS.map((option) => (
          <Button
            key={option.value}
            type="button"
            size="sm"
            variant={variant === option.value ? "primary" : "ghost"}
            onClick={() => onChange(option.value)}
          >
            {option.label}
          </Button>
        ))}
      </div>
    </div>
  );
}

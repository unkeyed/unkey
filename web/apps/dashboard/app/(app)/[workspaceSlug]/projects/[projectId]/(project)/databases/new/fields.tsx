"use client";

import {
  EnvironmentIcon,
  environmentName,
} from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/_components/environment-label";
import type { EnvironmentKind } from "@/lib/collections/deploy/environments";
import {
  FormField,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@unkey/ui";

export function PickerField<Value extends string>({
  label,
  value,
  options,
  placeholder,
  onChange,
}: {
  label: string;
  value: Value | null;
  options: { value: Value; label: string }[];
  placeholder?: string;
  onChange: (value: Value) => void;
}) {
  return (
    <FormField label={label}>
      {(control) => (
        <Select
          value={value}
          items={options}
          onValueChange={(next: Value | null) => (next ? onChange(next) : undefined)}
        >
          <SelectTrigger id={control.id} aria-describedby={control.describedBy}>
            <SelectValue placeholder={placeholder} />
          </SelectTrigger>
          <SelectContent>
            {options.map((option) => (
              <SelectItem key={option.value} value={option.value}>
                {option.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      )}
    </FormField>
  );
}

export function EnvironmentHeading({ env }: { env: EnvironmentKind }) {
  return (
    <>
      <EnvironmentIcon kind={env} className="size-4 text-gray-11" />
      <span className="font-medium text-gray-12">{environmentName(env)}</span>
    </>
  );
}

export function Price({ value }: { value: number }) {
  return <span className="font-medium text-gray-12 tabular-nums">${value}/mo</span>;
}

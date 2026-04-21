import { ChevronDown } from "@unkey/icons";
import * as React from "react";
import { cn } from "../../lib/utils";
import { FormDescription, FormLabel, type Requirement } from "./form-helpers";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "./select";

export type FormSelectOption = { value: string; label: React.ReactNode };

// Hack to populate fumadocs' AutoTypeTable
type DocumentedFormSelectProps = {
  label?: string;
  description?: string | React.ReactNode;
  requirement?: Requirement;
  error?: string;
  descriptionPosition?: "inline" | "label";
};

type FormSelectProps = DocumentedFormSelectProps & {
  options: FormSelectOption[];
  value: string;
  onValueChange: (value: string) => void;
  placeholder?: string;
  disabled?: boolean;
  id?: string;
  className?: string;
  triggerClassName?: string;
  contentClassName?: string;
  rightIcon?: React.ReactNode;
};

function FormSelect({
  label,
  description,
  error,
  requirement,
  id,
  className,
  triggerClassName,
  contentClassName,
  descriptionPosition = "inline",
  options,
  value,
  onValueChange,
  placeholder,
  disabled,
  rightIcon,
}: FormSelectProps) {
  const descriptionAsTooltip = descriptionPosition === "label";
  const selectVariant = error ? "error" : undefined;
  const selectId = id || React.useId();
  const descriptionId = `${selectId}-helper`;
  const errorId = `${selectId}-error`;

  return (
    <fieldset className={cn("flex flex-col gap-1.5 border-0 m-0 p-0", className)}>
      <FormLabel
        label={label}
        requirement={requirement}
        hasError={Boolean(error)}
        htmlFor={selectId}
        tooltipContent={descriptionAsTooltip ? description : undefined}
      />
      <Select value={value} onValueChange={onValueChange} disabled={disabled}>
        <SelectTrigger
          id={selectId}
          variant={selectVariant}
          className={triggerClassName}
          aria-describedby={error ? errorId : description ? descriptionId : undefined}
          aria-invalid={!!error}
          aria-required={requirement === "required"}
          rightIcon={rightIcon ?? <ChevronDown className="absolute right-2" iconSize="md-medium" />}
        >
          <SelectValue placeholder={placeholder} />
        </SelectTrigger>
        <SelectContent className={contentClassName}>
          {options.map((opt) => (
            <SelectItem key={opt.value} value={opt.value}>
              {opt.label}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
      <FormDescription
        description={descriptionAsTooltip ? undefined : description}
        error={error}
        descriptionId={descriptionId}
        errorId={errorId}
      />
    </fieldset>
  );
}

FormSelect.displayName = "FormSelect";

export { FormSelect, type FormSelectProps, type DocumentedFormSelectProps };

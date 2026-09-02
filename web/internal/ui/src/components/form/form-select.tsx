import { IconChevronDownOutline18 } from "nucleo-ui-outline-18";
// biome-ignore lint/style/useImportType: this package compiles JSX with the classic runtime, so React must stay a value import
import * as React from "react";
import { FormField } from "./form-field";
import type { Requirement } from "./form-helpers";
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
  return (
    <FormField
      label={label}
      description={description}
      error={error}
      requirement={requirement}
      id={id}
      className={className}
      descriptionPosition={descriptionPosition}
    >
      {(control) => (
        <Select
          // Base UI's Select.Value renders the raw value unless Root gets an
          // items map — without this, triggers show e.g. "basic_member" instead
          // of "Member".
          items={options.map((opt) => ({ value: opt.value, label: opt.label }))}
          value={value}
          onValueChange={(newValue) => {
            if (newValue !== null) {
              onValueChange(newValue);
            }
          }}
          disabled={disabled}
        >
          <SelectTrigger
            id={control.id}
            variant={control.variant}
            className={triggerClassName}
            aria-describedby={control.describedBy}
            aria-invalid={control.invalid}
            aria-required={requirement === "required"}
            rightIcon={
              rightIcon ?? <IconChevronDownOutline18 className="size-4 absolute right-2" />
            }
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
      )}
    </FormField>
  );
}

FormSelect.displayName = "FormSelect";

export { type DocumentedFormSelectProps, FormSelect, type FormSelectProps };

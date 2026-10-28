import { IconChevronDownOutline18 } from "@unkey/icons";
// biome-ignore lint/style/useImportType: this package compiles JSX with the classic runtime, so React must stay a value import
import * as React from "react";
import { FormField } from "./form-field";
import type { Requirement } from "./form-helpers";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "./select";

export type FormSelectOption<T extends string = string> = {
  value: T;
  label: React.ReactNode;
  disabled?: boolean;
};

// Hack to populate fumadocs' AutoTypeTable
type DocumentedFormSelectProps = {
  label?: string;
  description?: string | React.ReactNode;
  requirement?: Requirement;
  error?: string;
  descriptionPosition?: "inline" | "label";
};

type FormSelectValueProps<T extends string> =
  | { multiple?: false; value: T; onValueChange: (value: NoInfer<T>) => void }
  | { multiple: true; value: T[]; onValueChange: (value: NoInfer<T>[]) => void };

type FormSelectProps<T extends string = string> = DocumentedFormSelectProps &
  FormSelectValueProps<T> & {
    options: readonly FormSelectOption<T>[];
    placeholder?: string;
    disabled?: boolean;
    id?: string;
    size?: "md" | "sm";
    "aria-label"?: string;
    "aria-invalid"?: boolean;
    className?: string;
    triggerClassName?: string;
    valueClassName?: string;
    contentClassName?: string;
    itemClassName?: string;
    rightIcon?: React.ReactNode;
  };

function FormSelect<T extends string>(props: FormSelectProps<T>) {
  const {
    label,
    description,
    error,
    requirement,
    id,
    className,
    triggerClassName,
    valueClassName,
    contentClassName,
    itemClassName,
    descriptionPosition = "inline",
    options,
    placeholder,
    disabled,
    size,
    rightIcon,
  } = props;
  // Base UI's Select.Value renders the raw value unless Root gets an items
  // map — without this, triggers show e.g. "basic_member" instead of "Member".
  const items = options.map((opt) => ({ value: opt.value, label: opt.label }));

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
      {(control) => {
        const parts = (
          <>
            <SelectTrigger
              id={control.id}
              size={size}
              variant={control.variant}
              className={triggerClassName}
              aria-label={props["aria-label"]}
              aria-describedby={control.describedBy}
              aria-invalid={props["aria-invalid"] ?? control.invalid}
              aria-required={requirement === "required"}
              rightIcon={
                rightIcon ??
                (size === "sm" ? undefined : (
                  <IconChevronDownOutline18 className="size-3.5 absolute right-2" />
                ))
              }
            >
              <SelectValue placeholder={placeholder} className={valueClassName} />
            </SelectTrigger>
            <SelectContent className={contentClassName}>
              {options.map((opt) => (
                <SelectItem
                  key={opt.value}
                  value={opt.value}
                  disabled={opt.disabled}
                  className={itemClassName}
                >
                  {opt.label}
                </SelectItem>
              ))}
            </SelectContent>
          </>
        );
        return props.multiple ? (
          <Select
            multiple
            items={items}
            value={props.value}
            onValueChange={props.onValueChange}
            disabled={disabled}
          >
            {parts}
          </Select>
        ) : (
          <Select
            items={items}
            value={props.value}
            onValueChange={(next) => {
              if (next !== null) {
                props.onValueChange(next);
              }
            }}
            disabled={disabled}
          >
            {parts}
          </Select>
        );
      }}
    </FormField>
  );
}

FormSelect.displayName = "FormSelect";

export { FormSelect, type FormSelectProps, type DocumentedFormSelectProps };

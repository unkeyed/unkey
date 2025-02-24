import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { cn } from "@/lib/utils";
import { CircleInfo, TriangleWarning2 } from "@unkey/icons";
import * as React from "react";

export interface FormSelectProps {
  label?: string;
  description?: string;
  required?: boolean;
  error?: string;
  id?: string;
  className?: string;
  variant?: "default" | "success" | "warning" | "error";
  value: string;
  onChange: (value: string) => void;
  options: Array<{ value: string; label: string }>;
  placeholder?: string;
  disabled?: boolean;
  name?: string;
}

export const FormSelect = React.forwardRef<HTMLDivElement, FormSelectProps>(
  ({
    label,
    description,
    error,
    required,
    id,
    className,
    variant,
    value,
    onChange,
    options,
    placeholder,
    disabled,
    name,
    ...props
  }) => {
    const inputId = id || React.useId();
    const descriptionId = `${inputId}-helper`;
    const errorId = `${inputId}-error`;

    return (
      <fieldset className={cn("flex flex-col gap-1.5 border-0 m-0 p-0", className)}>
        {label && (
          <label
            id={`${inputId}-label`}
            htmlFor={inputId}
            className="text-gray-11 text-[13px] flex items-center"
          >
            {label}
            {required && (
              <span className="text-error-9 ml-1" aria-label="required field">
                *
              </span>
            )}
          </label>
        )}

        <Select onValueChange={onChange} value={value} disabled={disabled} name={name} {...props}>
          <SelectTrigger
            className="flex h-8 w-full items-center justify-between rounded-md bg-transparent px-3 py-2 text-[13px] border border-gray-4 focus:border focus:border-gray-4 hover:bg-gray-4 hover:border-gray-8 focus:bg-gray-4"
            id={inputId}
            aria-describedby={error ? errorId : description ? descriptionId : undefined}
            aria-invalid={!!error}
            aria-required={required}
          >
            <SelectValue placeholder={placeholder} />
          </SelectTrigger>
          <SelectContent className="border-none">
            {options.map((option) => (
              <SelectItem key={option.value} value={option.value}>
                {option.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>

        {(description || error) && (
          <div className="text-[13px] leading-5">
            {error ? (
              <div id={errorId} role="alert" className="text-error-11 flex gap-2 items-center">
                <TriangleWarning2 aria-hidden="true" />
                {error}
              </div>
            ) : description ? (
              <output
                id={descriptionId}
                className={cn(
                  "text-gray-9 flex gap-2 items-center",
                  variant === "success"
                    ? "text-success-11"
                    : variant === "warning"
                      ? "text-warning-11"
                      : "",
                )}
              >
                {variant === "warning" ? (
                  <TriangleWarning2 size="md-regular" aria-hidden="true" />
                ) : (
                  <CircleInfo size="md-regular" aria-hidden="true" />
                )}
                <span>{description}</span>
              </output>
            ) : null}
          </div>
        )}
      </fieldset>
    );
  },
);

FormSelect.displayName = "FormSelect";

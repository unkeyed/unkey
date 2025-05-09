"use client";

import { cn } from "@/lib/utils";
import { CircleInfo, TriangleWarning2 } from "@unkey/icons";
import { OptionalTag, RequiredTag } from "@unkey/ui/src/components/form";
import * as React from "react";
import { Combobox } from "./combobox";

// Documented props type for FormCombobox
export type DocumentedFormComboboxProps = {
  /**
   * The label text displayed above the combobox
   */
  label?: string;
  /**
   * Description text to provide additional context
   */
  description?: string | React.ReactNode;
  /**
   * Whether the field is required
   */
  required?: boolean;
  /**
   * Whether the field is optional (displays optional tag)
   */
  optional?: boolean;
  /**
   * Error message to display
   */
  error?: string;
};

// Props type combining Combobox props with form props
export type FormComboboxProps = React.ComponentProps<typeof Combobox> & DocumentedFormComboboxProps;

export const FormCombobox = React.forwardRef<HTMLDivElement, FormComboboxProps>(
  (
    {
      label,
      description,
      error,
      required,
      optional,
      className,
      wrapperClassName,
      variant,
      id: propId,
      ...props
    },
    ref,
  ) => {
    const inputVariant = error ? "error" : variant;
    const inputId = propId || React.useId();
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
            {required && <RequiredTag hasError={!!error} />}
            {optional && <OptionalTag />}
          </label>
        )}
        <div ref={ref}>
          <Combobox
            id={inputId}
            variant={inputVariant}
            wrapperClassName={wrapperClassName}
            aria-describedby={error ? errorId : description ? descriptionId : undefined}
            aria-invalid={!!error}
            aria-required={required}
            {...props}
          />
        </div>
        {(description || error) && (
          <div className="text-[13px] leading-5">
            {error ? (
              <div id={errorId} role="alert" className="text-error-11 flex gap-2 items-center">
                <TriangleWarning2 className="flex-shrink-0" aria-hidden="true" />
                <span className="flex-1">{error}</span>
              </div>
            ) : description ? (
              <output
                id={descriptionId}
                className={cn(
                  "text-gray-9 flex gap-2 items-start",
                  inputVariant === "success"
                    ? "text-success-11"
                    : inputVariant === "warning"
                      ? "text-warning-11"
                      : "",
                )}
              >
                <div className="size-[14px]">
                  {inputVariant === "warning" ? (
                    <TriangleWarning2
                      size="md-regular"
                      className="flex-shrink-0 mt-[3px]"
                      aria-hidden="true"
                    />
                  ) : (
                    <CircleInfo
                      size="md-regular"
                      className="flex-shrink-0 mt-[3px]"
                      aria-hidden="true"
                    />
                  )}
                </div>
                <span className="flex-1">{description}</span>
              </output>
            ) : null}
          </div>
        )}
      </fieldset>
    );
  },
);

FormCombobox.displayName = "FormCombobox";

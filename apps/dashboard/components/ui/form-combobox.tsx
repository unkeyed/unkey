"use client";

import { cn } from "@/lib/utils";
import { FormDescription } from "@unkey/ui/src/components/form/form-helpers";
import { OptionalTag, RequiredTag } from "@unkey/ui/src/components/form/form-tags";
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
  /**
   * Whether to show indicator for loading
   */
  loading?: boolean;
  /**
   * Tooltip text displayed on hover
   */
  title?: string;
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
          <FormDescription
            description={description}
            error={error}
            descriptionId={descriptionId}
            errorId={errorId}
          />
        )}
      </fieldset>
    );
  },
);

FormCombobox.displayName = "FormCombobox";

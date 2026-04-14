"use client";

import { cn } from "@/lib/utils";
import { CopyButton } from "@unkey/ui";
import { FormDescription, FormLabel } from "@unkey/ui/src/components/form/form-helpers";
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
  /**
   * When provided, shows a copy button next to the label that copies this value.
   * Typically used when the field is disabled/read-only.
   */
  copyValue?: string;
  /**
   * Where to render the description. "inline" (default) shows it below the
   * combobox; "label" shows it as a tooltip on an info icon next to the label.
   */
  descriptionPosition?: "inline" | "label";
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
      copyValue,
      id: propId,
      descriptionPosition = "inline",
      ...props
    },
    ref,
  ) => {
    const generatedId = React.useId();
    const inputVariant = error ? "error" : variant;
    const inputId = propId || generatedId;
    const descriptionId = `${inputId}-helper`;
    const errorId = `${inputId}-error`;
    const descriptionAsTooltip = descriptionPosition === "label";

    return (
      <fieldset className={cn("flex flex-col gap-1.5 border-0 m-0 p-0", className)}>
        <FormLabel
          label={label}
          required={required}
          optional={optional}
          hasError={!!error}
          htmlFor={inputId}
          tooltipContent={descriptionAsTooltip ? description : undefined}
        />
        <div ref={ref} className="relative">
          <Combobox
            id={inputId}
            variant={inputVariant}
            wrapperClassName={wrapperClassName}
            aria-describedby={error ? errorId : description ? descriptionId : undefined}
            aria-invalid={!!error}
            aria-required={required}
            {...props}
          />
          {copyValue && (
            <CopyButton
              value={copyValue}
              variant="ghost"
              className="absolute right-8 top-1/2 -translate-y-1/2 size-6 text-gray-12 [&_svg]:stroke-9"
              src="form-combobox"
            />
          )}
        </div>
        {((description && !descriptionAsTooltip) || error) && (
          <FormDescription
            description={descriptionAsTooltip ? undefined : description}
            error={error}
            variant={variant}
            descriptionId={descriptionId}
            errorId={errorId}
          />
        )}
      </fieldset>
    );
  },
);

FormCombobox.displayName = "FormCombobox";

"use client";

import { Combobox, CopyButton, FormField } from "@unkey/ui";
import type { Requirement } from "@unkey/ui/src/components/form/form-helpers";
import * as React from "react";

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
   * Error message to display
   */
  error?: string;
  /**
   * Whether the field is required or optional
   */
  requirement?: Requirement;
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
export type FormComboboxProps = React.ComponentProps<typeof Combobox> &
  DocumentedFormComboboxProps & {
    ref?: React.Ref<HTMLDivElement>;
  };

export function FormCombobox({
  label,
  description,
  error,
  requirement,
  className,
  wrapperClassName,
  variant,
  copyValue,
  id,
  descriptionPosition = "inline",
  ref,
  ...props
}: FormComboboxProps) {
  return (
    <FormField
      label={label}
      description={description}
      error={error}
      requirement={requirement}
      id={id}
      className={className}
      variant={variant}
      descriptionPosition={descriptionPosition}
    >
      {(control) => (
        <div ref={ref} className="relative">
          <Combobox
            id={control.id}
            variant={control.variant ?? variant}
            wrapperClassName={wrapperClassName}
            aria-describedby={control.describedBy}
            aria-invalid={control.invalid}
            aria-required={requirement === "required"}
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
      )}
    </FormField>
  );
}

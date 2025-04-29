import { CircleInfo, TriangleWarning2 } from "@unkey/icons";
import * as React from "react";
import { cn } from "../../lib/utils";
import { Checkbox, type CheckboxProps } from "../checkbox";
import { OptionalTag, RequiredTag } from "./form-textarea";

// Hack to populate fumadocs' AutoTypeTable
export type DocumentedFormCheckboxProps = {
  label?: string;
  description?: string | React.ReactNode;
  required?: boolean;
  optional?: boolean;
  error?: string;
  variant?: CheckboxProps["variant"];
  color?: CheckboxProps["color"];
  size?: CheckboxProps["size"];
};

export type FormCheckboxProps = Omit<CheckboxProps, "size" | "variant" | "color"> &
  DocumentedFormCheckboxProps;

export const FormCheckbox = React.forwardRef<HTMLButtonElement, FormCheckboxProps>(
  (
    {
      label,
      description,
      error,
      required,
      id,
      className,
      optional,
      variant = "primary",
      color,
      size = "md",
      ...props
    },
    ref,
  ) => {
    const checkboxVariant = error ? "primary" : variant;
    const checkboxColor = error ? "danger" : color;
    const checkboxId = id || React.useId();
    const descriptionId = `${checkboxId}-helper`;
    const errorId = `${checkboxId}-error`;

    return (
      <fieldset className={cn("flex flex-col gap-1.5 border-0 m-0 p-0", className)}>
        <div className="flex items-center gap-4">
          <Checkbox
            ref={ref}
            id={checkboxId}
            variant={checkboxVariant}
            color={checkboxColor}
            size={size}
            aria-describedby={error ? errorId : description ? descriptionId : undefined}
            aria-invalid={!!error}
            aria-required={required}
            {...props}
          />
          <div className="flex flex-col gap-1">
            {label && (
              <label
                id={`${checkboxId}-label`}
                htmlFor={checkboxId}
                className="text-gray-12 text-[13px] leading-5 flex items-center cursor-pointer"
              >
                {label}
                {required && <RequiredTag hasError={!!error} />}
                {optional && <OptionalTag />}
              </label>
            )}
          </div>
        </div>
        {(description || error) && (
          <div className="text-[13px] leading-5">
            {error ? (
              <div id={errorId} role="alert" className="text-error-11 flex gap-2 items-center">
                <TriangleWarning2 className="flex-shrink-0" size="sm-regular" aria-hidden="true" />
                <span className="flex-1">{error}</span>
              </div>
            ) : description ? (
              <output
                id={descriptionId}
                className={cn(
                  "text-gray-9 flex gap-2 items-start",
                  variant === "primary" && color === "success"
                    ? "text-success-11"
                    : variant === "primary" && color === "warning"
                      ? "text-warning-11"
                      : "",
                )}
              >
                <div className="size-[14px]">
                  {variant === "primary" && color === "warning" ? (
                    <TriangleWarning2
                      size="sm-regular"
                      className="flex-shrink-0 mt-[3px]"
                      aria-hidden="true"
                    />
                  ) : (
                    <CircleInfo
                      size="sm-regular"
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

FormCheckbox.displayName = "FormCheckbox";

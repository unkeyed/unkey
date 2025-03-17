import { CircleInfo, TriangleWarning2 } from "@unkey/icons";
import * as React from "react";
import { cn } from "../../lib/utils";
import { type DocumentedInputProps, Input, type InputProps } from "../input";
import { OptionalTag, RequiredTag } from "./form-textarea";

// Hack to populate fumadocs' AutoTypeTable
export type DocumentedFormInputProps = DocumentedInputProps & {
  label?: string;
  description?: string | React.ReactNode;
  required?: boolean;
  optional?: boolean;
  error?: string;
};

export type FormInputProps = InputProps & DocumentedFormInputProps;

export const FormInput = React.forwardRef<HTMLInputElement, FormInputProps>(
  ({ label, description, error, required, id, className, optional, variant, ...props }, ref) => {
    const inputVariant = error ? "error" : variant;
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
            {required && <RequiredTag hasError={!!error} />}
            {optional && <OptionalTag />}
          </label>
        )}
        <Input
          ref={ref}
          id={inputId}
          variant={inputVariant}
          aria-describedby={error ? errorId : description ? descriptionId : undefined}
          aria-invalid={!!error}
          aria-required={required}
          {...props}
        />
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
                  variant === "success"
                    ? "text-success-11"
                    : variant === "warning"
                      ? "text-warning-11"
                      : "",
                )}
              >
                {variant === "warning" ? (
                  <TriangleWarning2
                    size="md-regular"
                    className="flex-shrink-0"
                    aria-hidden="true"
                  />
                ) : (
                  <CircleInfo
                    size="md-regular"
                    className="flex-shrink-0 mt-[3px]"
                    aria-hidden="true"
                  />
                )}
                <span className="flex-1">{description}</span>
              </output>
            ) : null}
          </div>
        )}
      </fieldset>
    );
  },
);

FormInput.displayName = "FormInput";

import * as React from "react";
import { cn } from "../../lib/utils";
import { FormDescription, FormLabel } from "./form-helpers";
import { type DocumentedInputProps, Input, type InputProps } from "./input";

// Hack to populate fumadocs' AutoTypeTable
type DocumentedFormInputProps = DocumentedInputProps & {
  label?: string;
  description?: string | React.ReactNode;
  required?: boolean;
  optional?: boolean;
  error?: string;
};

type FormInputProps = InputProps & DocumentedFormInputProps;

const FormInput = React.forwardRef<HTMLInputElement, FormInputProps>(
  ({ label, description, error, required, id, className, optional, variant, ...props }, ref) => {
    const inputVariant = error ? "error" : variant;
    const inputId = id || React.useId();
    const descriptionId = `${inputId}-helper`;
    const errorId = `${inputId}-error`;

    return (
      <fieldset className={cn("flex flex-col gap-1.5 border-0 m-0 p-0", className)}>
        <FormLabel
          label={label}
          required={required}
          optional={optional}
          hasError={Boolean(error)}
          htmlFor={inputId}
        />
        <Input
          ref={ref}
          id={inputId}
          variant={inputVariant}
          aria-describedby={error ? errorId : description ? descriptionId : undefined}
          aria-invalid={!!error}
          aria-required={required}
          {...props}
        />
        <FormDescription
          description={description}
          error={error}
          variant={variant}
          descriptionId={descriptionId}
          errorId={errorId}
        />
      </fieldset>
    );
  },
);

FormInput.displayName = "FormInput";

export { FormInput, type FormInputProps, type DocumentedFormInputProps };

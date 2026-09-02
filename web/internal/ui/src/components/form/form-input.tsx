// biome-ignore lint/style/useImportType: this package compiles JSX with the classic runtime, so React must stay a value import
import * as React from "react";
import { FormField } from "./form-field";
import type { Requirement } from "./form-helpers";
import { type DocumentedInputProps, Input, type InputProps } from "./input";

// Hack to populate fumadocs' AutoTypeTable
type DocumentedFormInputProps = DocumentedInputProps & {
  label?: string;
  description?: string | React.ReactNode;
  requirement?: Requirement;
  error?: string;
  descriptionPosition?: "inline" | "label";
};

type FormInputProps = InputProps &
  DocumentedFormInputProps & {
    ref?: React.Ref<HTMLInputElement>;
  };

function FormInput({
  label,
  description,
  error,
  requirement,
  id,
  className,
  variant,
  descriptionPosition = "inline",
  ref,
  ...props
}: FormInputProps) {
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
        <Input
          ref={ref}
          id={control.id}
          variant={control.variant ?? variant}
          aria-describedby={control.describedBy}
          aria-invalid={control.invalid}
          aria-required={requirement === "required"}
          {...props}
        />
      )}
    </FormField>
  );
}

export { type DocumentedFormInputProps, FormInput, type FormInputProps };

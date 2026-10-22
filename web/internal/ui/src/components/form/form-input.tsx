// biome-ignore lint/style/useImportType: this package compiles JSX with the classic runtime, so React must stay a value import
import * as React from "react";
import { CopyButton } from "../buttons/copy-button";
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
  copyValue?: string;
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
  copyValue,
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
        <div className="relative">
          <Input
            ref={ref}
            id={control.id}
            className={copyValue ? "pr-10" : undefined}
            variant={control.variant ?? variant}
            aria-describedby={control.describedBy}
            aria-invalid={control.invalid}
            aria-required={requirement === "required"}
            {...props}
          />
          {copyValue && (
            <CopyButton
              value={copyValue}
              variant="ghost"
              className="absolute right-2 top-1/2 size-6 -translate-y-1/2 text-gray-12 [&_svg]:stroke-9"
              src="form-input"
            />
          )}
        </div>
      )}
    </FormField>
  );
}

export { FormInput, type FormInputProps, type DocumentedFormInputProps };

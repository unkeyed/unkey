import React from "react";
import { cn } from "../../lib/utils";
import { FormDescription, FormLabel } from "./form-helpers";
import { type DocumentedTextareaProps, Textarea, type TextareaProps } from "./textarea";

// Hack to populate fumadocs' AutoTypeTable
type DocumentedFormTextareaProps = DocumentedTextareaProps & {
  label?: string;
  description?: string | React.ReactNode;
  required?: boolean;
  optional?: boolean;
  error?: string;
};

type FormTextareaProps = TextareaProps & DocumentedFormTextareaProps;

const FormTextarea = React.forwardRef<HTMLTextAreaElement, FormTextareaProps>(
  (
    {
      label,
      description,
      error,
      required,
      optional,
      id,
      className,
      variant,
      leftIcon,
      rightIcon,
      wrapperClassName,
      ...props
    },
    ref,
  ) => {
    const textareaVariant = error ? "error" : variant;
    const textareaId = id || React.useId();
    const descriptionId = `${textareaId}-helper`;
    const errorId = `${textareaId}-error`;

    return (
      <fieldset className={cn("flex flex-col gap-1.5 border-0 m-0 p-0", className)}>
        <FormLabel
          label={label}
          required={required}
          optional={optional}
          hasError={!!error}
          htmlFor={textareaId}
        />

        <Textarea
          ref={ref}
          id={textareaId}
          variant={textareaVariant}
          leftIcon={leftIcon}
          rightIcon={rightIcon}
          wrapperClassName={wrapperClassName}
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

FormTextarea.displayName = "FormTextarea";

export { FormTextarea, type FormTextareaProps, type DocumentedFormTextareaProps };

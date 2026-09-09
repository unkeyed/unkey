import * as React from "react";
import { cn } from "../../lib/utils";
import { FormDescription, FormLabel, type Requirement } from "./form-helpers";

export type FormFieldControl = {
  id: string;
  describedBy: string | undefined;
  invalid: boolean;
  variant: "error" | undefined;
};

// Hack to populate fumadocs' AutoTypeTable
type DocumentedFormFieldProps = {
  label?: string;
  description?: string | React.ReactNode;
  requirement?: Requirement;
  error?: string;
  descriptionPosition?: "inline" | "label";
};

type FormFieldProps = DocumentedFormFieldProps & {
  id?: string;
  className?: string;
  variant?: string | null;
  children: (control: FormFieldControl) => React.ReactNode;
};

function FormField({
  label,
  description,
  error,
  requirement,
  id,
  className,
  variant,
  descriptionPosition = "inline",
  children,
}: FormFieldProps) {
  const descriptionAsTooltip = descriptionPosition === "label";
  const generatedId = React.useId();
  const fieldId = id || generatedId;
  const descriptionId = `${fieldId}-helper`;
  const errorId = `${fieldId}-error`;
  const inlineDescription = descriptionAsTooltip ? undefined : description;

  return (
    <fieldset className={cn("flex flex-col gap-1.5 border-0 m-0 p-0", className)}>
      <FormLabel
        label={label}
        requirement={requirement}
        hasError={Boolean(error)}
        htmlFor={fieldId}
        tooltipContent={descriptionAsTooltip ? description : undefined}
      />
      {children({
        id: fieldId,
        describedBy: error ? errorId : inlineDescription ? descriptionId : undefined,
        invalid: Boolean(error),
        variant: error ? "error" : undefined,
      })}
      <FormDescription
        description={inlineDescription}
        error={error}
        variant={variant}
        descriptionId={descriptionId}
        errorId={errorId}
      />
    </fieldset>
  );
}

export { type DocumentedFormFieldProps, FormField, type FormFieldProps };

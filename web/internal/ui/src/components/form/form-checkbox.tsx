import * as React from "react";
import { cn } from "../../lib/utils";
import { Checkbox, type CheckboxProps } from "./checkbox";
import { FormDescription, FormLabel, type Requirement } from "./form-helpers";

// Hack to populate fumadocs' AutoTypeTable
type DocumentedFormCheckboxProps = {
  label?: string;
  description?: string | React.ReactNode;
  requirement?: Requirement;
  error?: string;
  variant?: CheckboxProps["variant"];
  color?: CheckboxProps["color"];
  size?: CheckboxProps["size"];
  descriptionPosition?: "inline" | "label";
};

type FormCheckboxProps = Omit<CheckboxProps, "size" | "variant" | "color"> &
  DocumentedFormCheckboxProps;

const FormCheckbox = React.forwardRef<HTMLButtonElement, FormCheckboxProps>(
  (
    {
      label,
      description,
      error,
      requirement,
      id,
      className,
      variant = "primary",
      color,
      size = "md",
      descriptionPosition = "inline",
      ...props
    },
    ref,
  ) => {
    const descriptionAsTooltip = descriptionPosition === "label";
    const checkboxVariant = error ? "primary" : variant;
    const checkboxColor = error ? "danger" : color;
    const checkboxId = id || React.useId();
    const descriptionId = `${checkboxId}-helper`;
    const errorId = `${checkboxId}-error`;

    return (
      <fieldset className={cn("flex flex-col gap-1.5 border-0 m-0 p-0", className)}>
        <div className="flex items-center gap-2">
          <Checkbox
            ref={ref}
            id={checkboxId}
            variant={checkboxVariant}
            color={checkboxColor}
            size={size}
            aria-describedby={error ? errorId : description ? descriptionId : undefined}
            aria-invalid={Boolean(error)}
            aria-required={requirement === "required"}
            {...props}
          />
          {label && (
            <div className="flex flex-col gap-1">
              <FormLabel
                label={label}
                requirement={requirement}
                hasError={Boolean(error)}
                htmlFor={checkboxId}
                tooltipContent={descriptionAsTooltip ? description : undefined}
              />
            </div>
          )}
        </div>
        <FormDescription
          description={descriptionAsTooltip ? undefined : description}
          error={error}
          variant={color}
          descriptionId={descriptionId}
          errorId={errorId}
        />
      </fieldset>
    );
  },
);

FormCheckbox.displayName = "FormCheckbox";

export { FormCheckbox, type FormCheckboxProps, type DocumentedFormCheckboxProps };

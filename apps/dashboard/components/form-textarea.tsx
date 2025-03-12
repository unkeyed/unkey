import { CircleInfo, TriangleWarning2 } from "@unkey/icons";
import { cn } from "@unkey/ui/src/lib/utils";
import { type VariantProps, cva } from "class-variance-authority";
import * as React from "react";

const textareaVariants = cva(
  "flex min-h-9 w-full rounded-lg text-[13px] leading-5 transition-colors duration-300 disabled:cursor-not-allowed disabled:opacity-50 placeholder:text-gray-7 text-gray-12",
  {
    variants: {
      variant: {
        default: [
          "border border-gray-5 hover:border-gray-8 bg-gray-2",
          "focus:border focus:border-accent-12 focus:ring-4 focus:ring-gray-5 focus-visible:outline-none focus:ring-offset-0",
        ],
        success: [
          "border border-success-9 hover:border-success-10 bg-gray-2",
          "focus:border-success-8 focus:ring-2 focus:ring-success-2 focus-visible:outline-none",
        ],
        warning: [
          "border border-warning-9 hover:border-warning-10 bg-gray-2",
          "focus:border-warning-8 focus:ring-2 focus:ring-warning-2 focus-visible:outline-none",
        ],
        error: [
          "border border-error-9 hover:border-error-10 bg-gray-2",
          "focus:border-error-8 focus:ring-2 focus:ring-error-2 focus-visible:outline-none",
        ],
      },
    },
    defaultVariants: {
      variant: "default",
    },
  },
);

// Hack to populate fumadocs' AutoTypeTable
export type DocumentedFormTextareaProps = VariantProps<typeof textareaVariants> & {
  label?: string;
  description?: string;
  required?: boolean;
  optional?: boolean;
  error?: string;
  wrapperClassName?: string;
};

export type FormTextareaProps = DocumentedFormTextareaProps &
  React.TextareaHTMLAttributes<HTMLTextAreaElement>;

export const FormTextarea = React.forwardRef<HTMLTextAreaElement, FormTextareaProps>(
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
        {label && (
          <label
            id={`${textareaId}-label`}
            htmlFor={textareaId}
            className="text-gray-11 text-[13px] flex items-center gap-2"
          >
            {label}
            {required && (
              <span className="text-error-9 ml-1" aria-label="required field">
                *
              </span>
            )}
            {optional && (
              <span className="inline-flex items-center rounded border border-grayA-4 text-grayA-11 px-1 py-0.5 text-xs font-sans bg-grayA-3 ">
                Optional
              </span>
            )}
          </label>
        )}
        <textarea
          ref={ref}
          id={textareaId}
          className={cn(textareaVariants({ variant: textareaVariant }), "px-3 py-2")}
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

FormTextarea.displayName = "FormTextarea";

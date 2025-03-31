import { CircleInfo, TriangleWarning2 } from "@unkey/icons";
import React from "react";
import { cn } from "../../lib/utils";
import { type DocumentedTextareaProps, Textarea, type TextareaProps } from "../textarea";

// Hack to populate fumadocs' AutoTypeTable
export type DocumentedFormTextareaProps = DocumentedTextareaProps & {
  label?: string;
  description?: string | React.ReactNode;
  required?: boolean;
  optional?: boolean;
  error?: string;
};

export type FormTextareaProps = TextareaProps & DocumentedFormTextareaProps;

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
        {label && (
          <label
            id={`${textareaId}-label`}
            htmlFor={textareaId}
            className="text-gray-11 text-[13px] flex items-center"
          >
            {label}
            {required && <RequiredTag hasError={Boolean(error)} />}
            {optional && <OptionalTag />}
          </label>
        )}
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

type TagProps = {
  className?: string;
};

type RequiredTagProps = TagProps & {
  hasError?: boolean;
};

export const OptionalTag = ({ className }: TagProps) => {
  return (
    <span
      className={cn(
        "inline-flex items-center rounded border border-grayA-4 text-grayA-11 px-1 py-0.5 text-xs font-sans bg-grayA-3 ml-2",
        className,
      )}
    >
      Optional
    </span>
  );
};

export const RequiredTag = ({ className, hasError }: RequiredTagProps) => {
  return (
    <span
      className={cn(
        "inline-flex items-center rounded border px-1 py-0.5 text-xs font-sans ml-2",
        hasError
          ? "border-error-4 text-error-11 bg-error-3"
          : "border-warning-4 text-warning-11 bg-warning-3 dark:border-warning-4 dark:text-warning-11 dark:bg-warning-3",
        className,
      )}
    >
      Required
    </span>
  );
};

FormTextarea.displayName = "FormTextarea";

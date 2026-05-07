import type * as React from "react";
import type { FieldError as RHFFieldError } from "react-hook-form";
import { cn } from "~/lib/utils";

type FieldProps = React.HTMLAttributes<HTMLDivElement> & {
  "data-invalid"?: boolean;
};

export function Field({ className, ...props }: FieldProps) {
  return <div className={cn("flex flex-col gap-1.5", className)} {...props} />;
}

export function FieldGroup({ className, ...props }: React.HTMLAttributes<HTMLDivElement>) {
  return <div className={cn("flex flex-col gap-4", className)} {...props} />;
}

export function FieldLabel({ className, ...props }: React.LabelHTMLAttributes<HTMLLabelElement>) {
  // biome-ignore lint/a11y/noLabelWithoutControl: callers pass htmlFor; this is a generic label primitive
  return <label className={cn("font-medium text-gray-12 text-sm", className)} {...props} />;
}

export function FieldDescription({
  className,
  ...props
}: React.HTMLAttributes<HTMLParagraphElement>) {
  return <p className={cn("text-gray-11 text-xs", className)} {...props} />;
}

type FieldErrorProps = React.HTMLAttributes<HTMLParagraphElement> & {
  errors?: Array<RHFFieldError | undefined>;
};

export function FieldError({ className, errors, children, ...props }: FieldErrorProps) {
  const message = children ?? errors?.find((e) => e?.message)?.message;
  if (!message) {
    return null;
  }
  return (
    <p
      role="alert"
      aria-live="polite"
      className={cn("text-error-11 text-xs", className)}
      {...props}
    >
      {message}
    </p>
  );
}

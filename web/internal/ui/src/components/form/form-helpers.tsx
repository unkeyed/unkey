// Create a shared FormHelper component
import { CircleInfo, TriangleWarning2 } from "@unkey/icons";
// biome-ignore lint/style/useImportType: Reqired for silencing Biome
import React from "react";
import { cn } from "../../lib/utils";
import { OptionalTag, RequiredTag } from "./form-tags";

export type FormHelperProps = {
  description?: string | React.ReactNode;
  error?: string;
  variant?: string | null;
  descriptionId: string;
  errorId: string;
};

export const FormDescription = ({
  description,
  error,
  variant,
  descriptionId,
  errorId,
}: FormHelperProps) => {
  if (!description && !error) {
    return null;
  }

  return (
    <div className="text-[13px] leading-5">
      {error ? (
        <div id={errorId} role="alert" className="text-error-11 flex gap-2 items-center">
          <TriangleWarning2
            iconSize="md-medium"
            className="flex-shrink-0 ml-[-1px] mr-[1px]"
            aria-hidden="true"
          />
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
              iconSize="md-medium"
              className="flex-shrink-0 mt-[3px]"
              aria-hidden="true"
            />
          ) : (
            <CircleInfo
              iconSize="md-medium"
              className="flex-shrink-0 mt-[3px]"
              aria-hidden="true"
            />
          )}
          <span className="flex-1 text-gray-10">{description}</span>
        </output>
      ) : null}
    </div>
  );
};

export type FormLabelProps = {
  label?: string;
  required?: boolean;
  optional?: boolean;
  hasError?: boolean;
  htmlFor: string;
};

export const FormLabel = ({ label, required, optional, hasError, htmlFor }: FormLabelProps) => {
  if (!label) {
    return null;
  }

  return (
    <label
      id={`${htmlFor}-label`}
      htmlFor={htmlFor}
      className="text-gray-11 text-[13px] flex items-center"
    >
      {label}
      {required && <RequiredTag hasError={hasError} />}
      {optional && <OptionalTag />}
    </label>
  );
};

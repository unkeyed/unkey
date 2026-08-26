"use client";

import { CircleInfo } from "@unkey/icons";
import { FormInput, InfoTooltip, RequiredTag } from "@unkey/ui";
import type { ReactNode } from "react";
import { Controller, useFormContext, useFormState } from "react-hook-form";
import type { RootKeyFormValues } from "./schema";

type NamedFormValues = {
  name: string;
};

export function NameField() {
  const { control } = useFormContext<NamedFormValues>();

  return (
    <Controller
      control={control}
      name="name"
      render={({ field, fieldState }) => (
        <FormInput
          label="Name"
          requirement="required"
          placeholder="e.g. CI deploy key"
          ref={field.ref}
          value={field.value}
          onChange={field.onChange}
          error={fieldState.error?.message}
        />
      )}
    />
  );
}

type KeyFieldsProps = {
  children: ReactNode;
};

export function KeyFields({ children }: KeyFieldsProps) {
  const { control } = useFormContext<RootKeyFormValues>();
  const { errors } = useFormState({ control, name: "policies" });

  return (
    <div className="flex flex-col gap-6">
      <NameField />

      <div className="flex flex-col gap-2">
        <span className="flex h-5 items-center text-[13px] text-gray-11">
          Permissions
          <InfoTooltip
            content="Select the privileges you'd like this Root Key to have."
            position={{ side: "right" }}
          >
            <CircleInfo iconSize="sm-regular" className="ml-1.5 shrink-0 text-gray-9" />
          </InfoTooltip>
          <RequiredTag hasError={errors.policies !== undefined} />
        </span>
        {children}
      </div>
    </div>
  );
}

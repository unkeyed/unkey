"use client";

import { FormInput } from "@unkey/ui";
import { Controller, useFormContext, useWatch } from "react-hook-form";
import { connectionHostVariable } from "../../connection-rules";
import type { ConnectionFormValues } from "../../model/form";

export function VariableField({ savedName }: { savedName?: string }) {
  const { control } = useFormContext<ConnectionFormValues>();
  const name = useWatch({ control, name: "name" }).trim();
  const renamed = savedName !== undefined && name !== "" && name !== savedName;

  return (
    <div className="flex flex-col gap-2">
      <Controller
        control={control}
        name="name"
        render={({ field, fieldState }) => (
          <FormInput
            label="Name"
            value={field.value}
            onChange={field.onChange}
            onBlur={field.onBlur}
            error={fieldState.error?.message}
            description={
              name ? (
                <>
                  Available in your app as{" "}
                  <code className="font-mono text-gray-12">{connectionHostVariable(name)}</code>.
                </>
              ) : undefined
            }
          />
        )}
      />
      {renamed ? (
        <p className="text-xs text-warning-11">
          Renaming changes the variable. Update your code before you redeploy.
        </p>
      ) : null}
    </div>
  );
}

export function VariableSummary() {
  const { control } = useFormContext<ConnectionFormValues>();
  const name = useWatch({ control, name: "name" }).trim();
  return <code className="truncate font-mono text-xs">{connectionHostVariable(name)}</code>;
}

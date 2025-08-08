"use client";
import { FormInput } from "@unkey/ui";
import { Controller, useFormContext } from "react-hook-form";
import type { FormValues } from "../create-key.schema";
import { ExternalIdField } from "./external-id-field";

export const GeneralSetup = () => {
  const {
    register,
    formState: { errors },
    setValue,
    control,
  } = useFormContext<FormValues>();

  return (
    <div className="space-y-5 px-2 py-1 ">
      <FormInput
        className="[&_input:first-of-type]:h-[36px]"
        placeholder="Enter name"
        label="Name"
        maxLength={256}
        description="Optional name to help identify this particular key."
        error={errors.name?.message}
        variant="default"
        optional
        {...register("name")}
      />
      <FormInput
        className="[&_input:first-of-type]:h-[36px]"
        label="Prefix"
        placeholder="Enter prefix"
        maxLength={8}
        description="Prefix to distinguish between different APIs (we'll add the underscore)."
        error={errors.prefix?.message}
        optional
        {...register("prefix")}
      />

      <Controller
        name="identityId"
        control={control}
        defaultValue=""
        render={({ field }) => (
          <ExternalIdField
            value={field.value ?? null}
            onChange={(identityId: string | null, externalId: string | null) => {
              field.onChange(identityId);
              setValue("externalId", externalId);
            }}
            error={errors.externalId?.message}
          />
        )}
      />

      <FormInput
        className="[&_input:first-of-type]:h-[36px]"
        label="Bytes"
        placeholder="Enter bytes"
        type="number"
        description="Key length in bytes - longer keys are more secure."
        error={errors.bytes?.message}
        optional
        min={8}
        max={255}
        maxLength={3}
        {...register("bytes")}
      />
    </div>
  );
};

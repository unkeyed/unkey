"use client";

import { FormInput } from "@unkey/ui";
import { useFormContext } from "react-hook-form";
import type { FormValues } from "./create-identity.schema";

export const GeneralSetup = () => {
  const {
    register,
    formState: { errors },
  } = useFormContext<FormValues>();

  return (
    <div className="space-y-5 px-2 py-1">
      <FormInput
        label="External ID"
        description="A unique identifier for this identity (3-255 characters)"
        error={errors.externalId?.message}
        {...register("externalId")}
        placeholder="user_123 or user@example.com"
        data-1p-ignore
        required
      />
    </div>
  );
};

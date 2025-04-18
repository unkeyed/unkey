"use client";
import { FormInput } from "@unkey/ui";
import { useFormContext } from "react-hook-form";
import type { FormValues } from "../schema";

export const GeneralSetup = () => {
  const {
    register,
    formState: { errors },
  } = useFormContext<FormValues>();

  return (
    <div className="space-y-5 px-2 py-1 ">
      <FormInput
        className="[&_input:first-of-type]:h-[36px]"
        placeholder="Enter name"
        label="Identifier"
        description="Optional name to help identify this particular key."
        error={errors.name?.message}
        optional
        {...register("name")}
      />
      <FormInput
        className="[&_input:first-of-type]:h-[36px]"
        label="Prefix"
        placeholder="Enter prefix"
        description="Prefix to distinguish between different APIs (we'll add the underscore)."
        error={errors.prefix?.message}
        optional
        {...register("prefix")}
      />
      <FormInput
        className="[&_input:first-of-type]:h-[36px]"
        label="Owner ID"
        placeholder="Enter owner ID"
        description="ID of the user/workspace in your system for key attribution."
        error={errors.ownerId?.message}
        optional
        {...register("ownerId")}
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
        {...register("bytes")}
      />
      <FormInput
        className="[&_input:first-of-type]:h-[36px]"
        label="Environments"
        placeholder="Enter environment (e.g. test, dev, prod)"
        description="Environment label to separate keys (e.g. test, live)."
        error={errors.environment?.message}
        optional
        {...register("environment")}
      />
    </div>
  );
};

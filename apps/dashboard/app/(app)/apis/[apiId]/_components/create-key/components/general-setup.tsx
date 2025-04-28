"use client";
import { FormInput } from "@unkey/ui";
import { useFormContext } from "react-hook-form";
import type { FormValues } from "../create-key.schema";

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
        label="Name"
        maxLength={256}
        description="Optional name to help identify this particular key."
        error={errors.name?.message}
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
      <FormInput
        className="[&_input:first-of-type]:h-[36px]"
        label="External ID"
        maxLength={256}
        placeholder="Enter external ID"
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
        maxLength={3}
        {...register("bytes")}
      />

      {/* INFO: We'll enable that soon */}
      {/* <FormInput */}
      {/*   className="[&_input:first-of-type]:h-[36px]" */}
      {/*   label="Environments" */}
      {/*   maxLength={256} */}
      {/*   placeholder="Enter environment (e.g. test, dev, prod)" */}
      {/*   description="Environment label to separate keys (e.g. test, live)." */}
      {/*   error={errors.environment?.message} */}
      {/*   optional */}
      {/*   {...register("environment")} */}
      {/* /> */}
    </div>
  );
};

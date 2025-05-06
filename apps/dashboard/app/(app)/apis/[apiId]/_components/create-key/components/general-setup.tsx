"use client";
import { FormCombobox } from "@/components/ui/form-combobox";
import { FormInput } from "@unkey/ui";
import { useState } from "react";
import { useFormContext } from "react-hook-form";
import type { FormValues } from "../create-key.schema";
import { useFetchIdentities } from "../hooks/use-fetch-identities";
import { createIdentityOptions } from "../hooks/use-fetch-identities/create-identity-options";

export const GeneralSetup = () => {
  const {
    register,
    formState: { errors },
    setValue,
  } = useFormContext<FormValues>();

  const [selectedIdentityId, setSelectedIdentityId] = useState<string | null>(null);

  const { identities, isFetchingNextPage, hasNextPage, loadMore } = useFetchIdentities();

  const identityOptions = createIdentityOptions({
    identities,
    hasNextPage,
    isFetchingNextPage,
    loadMore,
  });

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
      <FormCombobox
        optional
        label="External ID"
        description="ID of the user/workspace in your system for key attribution."
        options={identityOptions}
        value={selectedIdentityId || ""}
        onSelect={(val) => {
          const identity = identities.find((id) => id.id === val);
          setSelectedIdentityId(identity?.id || null);
          setValue("externalId", identity?.id || "");
        }}
        placeholder={
          <div className="flex w-full text-grayA-8 text-xs gap-1.5 items-center py-2">
            Select external ID
          </div>
        }
        searchPlaceholder="Search external ID..."
        emptyMessage="No external ID found."
        variant="default"
        error={errors.externalId?.message}
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

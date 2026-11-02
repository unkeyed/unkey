"use client";

import { SUPPORT_MAILTO } from "@/lib/support";
import { IconCircleInfoOutline18 } from "@unkey/icons";
import { FormInput, FormTextarea, InfoTooltip, RequiredTag } from "@unkey/ui";
import type { ReactNode } from "react";
import { Controller, useFormContext, useFormState } from "react-hook-form";
import type { RootKeyFormValues } from "./schema";

type NamedFormValues = {
  name: string;
};

function NameField() {
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
  nameHidden?: boolean;
  ipAllowlistEnabled?: boolean;
};

export function KeyFields({ children, nameHidden = false, ipAllowlistEnabled }: KeyFieldsProps) {
  const { control } = useFormContext<RootKeyFormValues>();
  const { errors } = useFormState({ control, name: "policies" });

  return (
    <div className="flex flex-col gap-6">
      {nameHidden ? null : <NameField />}

      {ipAllowlistEnabled !== undefined && (
        <Controller
          control={control}
          name="ipAllowlist"
          render={({ field, fieldState }) => (
            <FormTextarea
              {...field}
              label="IP allowlist"
              requirement="optional"
              rows={4}
              placeholder={"203.0.113.10\n198.51.100.0/24\n2001:db8::/32"}
              description={
                ipAllowlistEnabled ? (
                  "Enter one IP address or CIDR range per line. Leave empty to allow any IP address. This restricts callers that use this root key, not your end users."
                ) : (
                  <>
                    {field.value
                      ? "This allowlist is still enforced. "
                      : "Restrict this root key to specific IP addresses or CIDR ranges. "}
                    <a href={SUPPORT_MAILTO} className="underline underline-offset-2">
                      Contact support
                    </a>
                    {" to enable IP allowlist changes for this workspace."}
                  </>
                )
              }
              disabled={!ipAllowlistEnabled}
              error={fieldState.error?.message}
            />
          )}
        />
      )}

      <div className="flex flex-col gap-2">
        <span className="flex h-5 items-center text-sm text-gray-11">
          Permissions
          <InfoTooltip
            content="Select the privileges you'd like this Root Key to have."
            position={{ side: "right" }}
          >
            <IconCircleInfoOutline18 className="ml-1.5 size-3.5 shrink-0 text-gray-9" />
          </InfoTooltip>
          <RequiredTag hasError={errors.policies !== undefined} />
        </span>
        {children}
      </div>
    </div>
  );
}

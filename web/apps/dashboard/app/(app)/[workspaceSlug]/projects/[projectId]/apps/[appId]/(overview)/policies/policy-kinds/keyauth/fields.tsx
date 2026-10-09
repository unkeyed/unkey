"use client";

import { FormInput } from "@unkey/ui";
import { useController, useFormContext, useWatch } from "react-hook-form";
import { CreditCostField } from "./credits-field";
import { KeyspacesField } from "./keyspaces-field";
import { KeyLocationField } from "./location-field";
import { type KeyauthFormValues, keyauthSummary } from "./model";
import { KeyRatelimitsField } from "./ratelimits-field";
import { useKeyspaceNames } from "./use-keyspace-names";

export function KeyAuthSummary() {
  const { control, getValues } = useFormContext<KeyauthFormValues>();
  useWatch({ control });
  const keyspaceNames = useKeyspaceNames();
  return keyauthSummary(getValues(), keyspaceNames);
}

export function KeyAuthFields() {
  return (
    <div className="flex flex-col gap-5">
      <KeyspacesField />
      <KeyLocationField />
      <PermissionQueryField />
      <CreditCostField />
      <KeyRatelimitsField />
    </div>
  );
}

function PermissionQueryField() {
  const { control } = useFormContext<KeyauthFormValues>();
  const { field } = useController({ control, name: "permissionQuery" });
  return (
    <FormInput
      label="Permission query (optional)"
      placeholder="e.g. api.read AND api.write"
      value={field.value}
      onChange={(e) => field.onChange(e.target.value)}
      descriptionPosition="inline"
      description={
        <span>
          Supports <span className="text-gray-12 font-medium">AND</span> /{" "}
          <span className="text-gray-12 font-medium">OR</span> operators.
        </span>
      }
    />
  );
}

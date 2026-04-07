import { match } from "@unkey/match";
import type { Control } from "react-hook-form";
import { KeyAuthFields } from "./forms/keyauth-fields";
import { RateLimitFields } from "./forms/ratelimit-fields";
import type { PolicyFormValues, PolicyType } from "./schema";

export function PolicyConfigFields({
  type,
  control,
}: {
  type: PolicyType;
  control: Control<PolicyFormValues>;
}) {
  return match(type)
    .with("keyauth", () => (
      <KeyAuthFields control={control as Control<Extract<PolicyFormValues, { type: "keyauth" }>>} />
    ))
    .with("ratelimit", () => (
      <RateLimitFields
        control={control as Control<Extract<PolicyFormValues, { type: "ratelimit" }>>}
      />
    ))
    .exhaustive();
}

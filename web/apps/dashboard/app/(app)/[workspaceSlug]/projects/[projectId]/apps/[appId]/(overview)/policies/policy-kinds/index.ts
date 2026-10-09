import type { Policy, PolicyInput, PolicyType } from "@/lib/collections/deploy/policies.schema";
import type { FieldErrors } from "react-hook-form";
import { z } from "zod";
import { fromMatchExpr, toMatchExpr } from "./conditions/schema";
import { firewall, firewallFormSchema } from "./firewall/model";
import { keyauth, keyauthFormSchema } from "./keyauth/model";
import { logging, loggingFormSchema } from "./logging/model";
import { openapi, openapiFormSchema } from "./openapi/model";
import { ratelimit, ratelimitFormSchema } from "./ratelimit/model";
import { sharedFormFields } from "./shared";
import type { PolicyKindModel } from "./types";

export type { PolicyType } from "@/lib/collections/deploy/policies.schema";

export const policyFormSchema = z.discriminatedUnion("type", [
  keyauthFormSchema,
  ratelimitFormSchema,
  firewallFormSchema,
  openapiFormSchema,
  loggingFormSchema,
]);
export type PolicyFormValues = z.infer<typeof policyFormSchema>;

type FormMap = { [T in PolicyType]: Extract<PolicyFormValues, { type: T }> };

export const POLICY_KINDS: { [T in PolicyType]: PolicyKindModel<T, FormMap[T]> } = {
  keyauth,
  ratelimit,
  firewall,
  openapi,
  logging,
};

function isPolicyType(key: string): key is PolicyType {
  return Object.hasOwn(POLICY_KINDS, key);
}

export const POLICY_TYPES: readonly PolicyType[] = Object.keys(POLICY_KINDS).filter(isPolicyType);

export function getDefaultValues(type: PolicyType): PolicyFormValues {
  return {
    name: "",
    matchConditions: [],
    ...POLICY_KINDS[type].defaults(),
  };
}

// Indexing with a union type loses the link between a kind and its form
// values; a generic lookup keeps it.
function kindOf<T extends PolicyType>(type: T): PolicyKindModel<T, FormMap[T]> {
  return POLICY_KINDS[type];
}

export function summarize(values: PolicyFormValues): string {
  return kindOf(values.type).summary(values);
}

export function toPolicy(values: PolicyFormValues): PolicyInput {
  return {
    name: values.name,
    enabled: true,
    match: values.matchConditions.map(toMatchExpr),
    ...kindOf(values.type).toWire(values),
  };
}

export function fromPolicy(policy: Policy): PolicyFormValues {
  return {
    name: policy.name,
    matchConditions: (policy.match ?? []).map(fromMatchExpr).filter((c) => c !== null),
    ...kindOf(policy.type).fromWire(policy),
  };
}

const SHARED_FIELDS: ReadonlySet<string> = new Set([...Object.keys(sharedFormFields), "type"]);

export function hasConfigErrors(errors: FieldErrors<PolicyFormValues>): boolean {
  return Object.keys(errors).some((key) => !SHARED_FIELDS.has(key));
}

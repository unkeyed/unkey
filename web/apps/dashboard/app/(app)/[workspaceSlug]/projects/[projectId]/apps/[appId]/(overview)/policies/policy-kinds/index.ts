import type { Policy, PolicyType } from "@/lib/collections/deploy/policies.schema";
import { newUid } from "@unkey/id";
import type { FieldErrors } from "react-hook-form";
import { z } from "zod";
import { fromMatchExpr, toMatchExpr } from "../components/policy-form/rule/condition-schema";
import { firewall, firewallFormSchema } from "./firewall/model";
import { keyauth, keyauthFormSchema } from "./keyauth/model";
import { logging, loggingFormSchema } from "./logging/model";
import { openapi, openapiFormSchema } from "./openapi/model";
import { ratelimit, ratelimitFormSchema } from "./ratelimit/model";
import { sharedFormFields } from "./shared";
import type { KindFields, PolicyKindModel, WireBody, WireOf } from "./types";

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
type WireMap = { [T in PolicyType]: WireOf<T> };

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

export const ALL_ENVIRONMENTS = "__all__";

export function getDefaultValues(type: PolicyType): PolicyFormValues {
  return {
    name: "",
    environmentId: ALL_ENVIRONMENTS,
    matchConditions: [],
    ...POLICY_KINDS[type].defaults(),
  };
}

function toWireBody<T extends PolicyType>(type: T, values: FormMap[T]): WireBody<T> {
  return POLICY_KINDS[type].toWire(values);
}

function fromWireBody<T extends PolicyType>(type: T, policy: WireMap[T]): KindFields<FormMap[T]> {
  return POLICY_KINDS[type].fromWire(policy);
}

export function toPolicy(values: PolicyFormValues, existingId?: string): Policy {
  return {
    id: existingId ?? newUid("policy"),
    name: values.name,
    enabled: true,
    match: values.matchConditions.map(toMatchExpr),
    ...toWireBody(values.type, values),
  };
}

export function fromPolicy(policy: Policy): PolicyFormValues {
  return {
    name: policy.name,
    environmentId: ALL_ENVIRONMENTS,
    matchConditions: (policy.match ?? []).map(fromMatchExpr).filter((c) => c !== null),
    ...fromWireBody(policy.type, policy),
  };
}

const SHARED_FIELDS: ReadonlySet<string> = new Set([...Object.keys(sharedFormFields), "type"]);

export function hasConfigErrors(errors: FieldErrors<PolicyFormValues>): boolean {
  return Object.keys(errors).some((key) => !SHARED_FIELDS.has(key));
}

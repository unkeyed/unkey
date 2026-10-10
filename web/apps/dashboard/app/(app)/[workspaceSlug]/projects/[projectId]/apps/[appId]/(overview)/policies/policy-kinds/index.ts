import type { Policy, PolicyInput, PolicyType } from "@/lib/collections/deploy/policies.schema";
import type { FieldErrors } from "react-hook-form";
import { z } from "zod";
import { fromMatchExpr, toMatchExpr } from "./conditions/schema";
import { firewall } from "./firewall/model";
import { keyauth } from "./keyauth/model";
import { logging } from "./logging/model";
import { openapi } from "./openapi/model";
import { ratelimit } from "./ratelimit/model";
import type { PolicyKindModel, SummaryContext } from "./types";

export type { PolicyType } from "@/lib/collections/deploy/policies.schema";

const KIND_MODELS = {
  keyauth,
  ratelimit,
  firewall,
  openapi,
  logging,
} satisfies Record<PolicyType, unknown>;

function isPolicyType(key: string): key is PolicyType {
  return Object.hasOwn(KIND_MODELS, key);
}

export const POLICY_TYPES: readonly PolicyType[] = Object.keys(KIND_MODELS).filter(isPolicyType);

function nonEmpty<T>(items: readonly T[]): [T, ...T[]] {
  const [first, ...rest] = items;
  if (first === undefined) {
    throw new Error("no policy kinds");
  }
  return [first, ...rest];
}

export const policyFormSchema = z.discriminatedUnion(
  "type",
  nonEmpty(POLICY_TYPES.map((type) => KIND_MODELS[type].formSchema)),
);
export type PolicyFormValues = z.infer<typeof policyFormSchema>;

type FormMap = { [T in PolicyType]: Extract<PolicyFormValues, { type: T }> };

export const POLICY_KINDS: { [T in PolicyType]: PolicyKindModel<T, FormMap[T]> } = KIND_MODELS;

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

export function summarize(values: PolicyFormValues, context: SummaryContext): string {
  return kindOf(values.type).summary(values, context);
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
    matchConditions: (policy.match ?? []).map(fromMatchExpr),
    ...kindOf(policy.type).fromWire(policy),
  };
}

export function hasConfigErrors(type: PolicyType, errors: FieldErrors<PolicyFormValues>): boolean {
  return Object.keys(POLICY_KINDS[type].defaults()).some((field) => field in errors);
}

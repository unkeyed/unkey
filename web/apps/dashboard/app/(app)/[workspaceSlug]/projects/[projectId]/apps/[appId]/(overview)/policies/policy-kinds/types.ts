import type { Policy, PolicyType } from "@/lib/collections/deploy/policies.schema";
import type { IconKey2Outline18 } from "@unkey/icons";
import type { SharedFormField } from "./shared";

export const POLICY_DOCS_URL = "https://www.unkey.com/docs/compute/gateway/policies";

export type Rejection = { status: 400 | 401 | 403 | 429; reason: string };

const STATUS_NAMES: Record<Rejection["status"], string> = {
  400: "Bad Request",
  401: "Unauthorized",
  403: "Forbidden",
  429: "Too Many Requests",
};

export function statusLabel(status: Rejection["status"]): string {
  return `${STATUS_NAMES[status]} (${status})`;
}

export type WireOf<T extends PolicyType> = Extract<Policy, { type: T }>;

export type WireBody<T extends PolicyType> = T extends PolicyType
  ? Omit<WireOf<T>, "id" | "name" | "enabled" | "match">
  : never;

export type KindFields<Form> = Form extends unknown ? Omit<Form, SharedFormField> : never;

export type PolicyKindModel<T extends PolicyType, Form> = {
  label: string;
  Icon: typeof IconKey2Outline18;
  does: string;
  rejects: readonly Rejection[];
  defaults: () => KindFields<Form>;
  toWire: (values: Form) => WireBody<T>;
  fromWire: (policy: WireOf<T>) => KindFields<Form>;
  summary: (values: KindFields<Form>) => string;
};

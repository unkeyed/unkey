"use client";

import type { PolicyType } from "@/lib/collections/deploy/policies.schema";
import type { ComponentType } from "react";
import { useFormContext, useWatch } from "react-hook-form";
import { type PolicyFormValues, summarize } from ".";
import { KeyAuthFields, KeyAuthSummary } from "./keyauth/fields";
import { LoggingFields } from "./logging/fields";
import { OpenApiFields } from "./openapi/fields";
import { RateLimitFields } from "./ratelimit/fields";

type KindUi = {
  /** The kind's config, shown when the rule's Then row expands. Null when it has none. */
  Fields: ComponentType | null;
  /** Replaces the model's summary when the line needs looked-up data. */
  Summary?: ComponentType;
};

export const POLICY_KIND_UI: Record<PolicyType, KindUi> = {
  keyauth: { Fields: KeyAuthFields, Summary: KeyAuthSummary },
  ratelimit: { Fields: RateLimitFields },
  firewall: { Fields: null },
  openapi: { Fields: OpenApiFields },
  logging: { Fields: LoggingFields },
};

function ModelSummary() {
  const { control, getValues } = useFormContext<PolicyFormValues>();
  // useWatch re-renders on every change; getValues gives the typed snapshot.
  useWatch({ control });
  return summarize(getValues());
}

export function KindSummary({ type }: { type: PolicyType }) {
  const Summary = POLICY_KIND_UI[type].Summary ?? ModelSummary;
  return <Summary />;
}

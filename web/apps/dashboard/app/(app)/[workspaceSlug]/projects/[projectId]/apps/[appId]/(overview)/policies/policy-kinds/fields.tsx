"use client";

import type { PolicyType } from "@/lib/collections/deploy/policies.schema";
import type { ComponentType } from "react";
import { useFormContext, useWatch } from "react-hook-form";
import { type PolicyFormValues, summarize } from ".";
import { KeyAuthFields } from "./keyauth/fields";
import { useKeyspaceNames } from "./keyauth/use-keyspace-names";
import { LoggingFields } from "./logging/fields";
import { OpenApiFields } from "./openapi/fields";
import { RateLimitFields } from "./ratelimit/fields";

export const POLICY_KIND_FIELDS: Record<PolicyType, ComponentType | null> = {
  keyauth: KeyAuthFields,
  ratelimit: RateLimitFields,
  firewall: null,
  openapi: OpenApiFields,
  logging: LoggingFields,
};

export function KindSummary() {
  const { control, getValues } = useFormContext<PolicyFormValues>();
  // useWatch re-renders on every change; getValues gives the typed snapshot.
  useWatch({ control });
  const keyspaceNames = useKeyspaceNames();
  return summarize(getValues(), { keyspaceNames });
}

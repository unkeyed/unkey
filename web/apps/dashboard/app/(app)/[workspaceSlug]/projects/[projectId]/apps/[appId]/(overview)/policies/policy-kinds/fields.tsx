"use client";

import type { PolicyType } from "@/lib/collections/deploy/policies.schema";
import type { ComponentType } from "react";
import { FIREWALL_SUMMARY } from "./firewall/model";
import { KeyAuthFields, KeyauthSummary } from "./keyauth/fields";
import { LoggingFields, LoggingSummary } from "./logging/fields";
import { OpenApiFields } from "./openapi/fields";
import { OPENAPI_SUMMARY } from "./openapi/model";
import { RateLimitFields, RatelimitSummary } from "./ratelimit/fields";

type PolicyKindFields = {
  /** The kind's config, shown when the rule's Then row expands. Null when it has none. */
  Fields: ComponentType | null;
  Summary: ComponentType;
};

export const POLICY_KIND_FIELDS: Record<PolicyType, PolicyKindFields> = {
  keyauth: { Fields: KeyAuthFields, Summary: KeyauthSummary },
  ratelimit: { Fields: RateLimitFields, Summary: RatelimitSummary },
  firewall: { Fields: null, Summary: () => FIREWALL_SUMMARY },
  openapi: { Fields: OpenApiFields, Summary: () => OPENAPI_SUMMARY },
  logging: { Fields: LoggingFields, Summary: LoggingSummary },
};

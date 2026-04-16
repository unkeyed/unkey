"use client";

import { FormInput, FormSelect } from "@unkey/ui";
import type React from "react";
import { useController, useFormContext, useWatch } from "react-hook-form";
import { DocsLink, Sep, Strong } from "./summary-helpers";

export type RateLimitIdentifierSource =
  | "remoteIp"
  | "header"
  | "authenticatedSubject"
  | "path"
  | "principalField";

type RatelimitFormValues = {
  type: "ratelimit";
  name: string;
  environmentId: string;
  limit: number;
  windowMs: number;
  identifierSource: RateLimitIdentifierSource;
  identifierValue: string;
};

const IDENTIFIER_SOURCE_LABELS: Record<RateLimitIdentifierSource, string> = {
  remoteIp: "IP",
  header: "Header",
  authenticatedSubject: "Subject",
  path: "Path",
  principalField: "Field",
};

const RATE_LIMIT_DOCS_URL =
  "https://www.unkey.com/docs/platform/sentinel/policies/rate-limiting#rate-limit-subjects";

const IDENTIFIER_SOURCE_DESCRIPTIONS: Record<RateLimitIdentifierSource, React.ReactNode> = {
  remoteIp: "Limit by client IP address.",
  header: "Limit by a specific request header (e.g. X-Tenant-Id).",
  authenticatedSubject: (
    <>
      Limit by the authenticated Principal's subject field. <DocsLink href={RATE_LIMIT_DOCS_URL} />
    </>
  ),
  path: "Create separate limits per endpoint.",
  principalField: (
    <>
      Limit by a field from the Principal's source (e.g. source.key.meta.org_id for per-organization
      limits). <DocsLink href={RATE_LIMIT_DOCS_URL} />
    </>
  ),
};

const IDENTIFIER_SOURCE_OPTIONS: { value: RateLimitIdentifierSource; label: string }[] = [
  { value: "remoteIp", label: "Remote IP" },
  { value: "header", label: "Header" },
  { value: "authenticatedSubject", label: "Authenticated Subject" },
  { value: "path", label: "Request Path" },
  { value: "principalField", label: "Principal Field" },
];

export function RateLimitFields() {
  const { control } = useFormContext<RatelimitFormValues>();

  const {
    field: { value: limit, onChange: onLimitChange },
    fieldState: { error: limitError },
  } = useController({ control, name: "limit" });

  const {
    field: { value: windowMs, onChange: onWindowChange },
    fieldState: { error: windowError },
  } = useController({ control, name: "windowMs" });

  const {
    field: { value: identifierSource, onChange: onIdentifierSourceChange },
  } = useController({ control, name: "identifierSource" });

  const {
    field: { value: identifierValue, onChange: onIdentifierValueChange },
  } = useController({ control, name: "identifierValue" });

  const needsIdentifierValue =
    identifierSource === "header" || identifierSource === "principalField";

  return (
    <div className="flex flex-col gap-4">
      <div className="flex gap-3">
        <FormInput
          label="Limit"
          descriptionPosition="label"
          description="Max number of requests allowed per window."
          type="number"
          value={limit}
          onChange={(e) => onLimitChange(Number.parseInt(e.target.value) || 0)}
          className="flex-1"
          error={limitError?.message}
        />
        <FormInput
          label="Window (ms)"
          type="number"
          value={windowMs}
          onChange={(e) => onWindowChange(Number.parseInt(e.target.value) || 0)}
          className="flex-1"
          descriptionPosition="label"
          description={
            windowMs > 0
              ? `${(windowMs / 1000).toFixed(1)}s. Time window before the counter resets.`
              : "Time window before the counter resets."
          }
          error={windowError?.message}
        />
      </div>

      <FormSelect
        label="Identifier"
        options={IDENTIFIER_SOURCE_OPTIONS}
        value={identifierSource}
        onValueChange={(v) => {
          onIdentifierSourceChange(v as RateLimitIdentifierSource);
          onIdentifierValueChange("");
        }}
        description={IDENTIFIER_SOURCE_DESCRIPTIONS[identifierSource]}
      />

      {needsIdentifierValue && (
        <FormInput
          label={identifierSource === "header" ? "Header Name" : "Field Path"}
          value={identifierValue}
          placeholder={identifierSource === "header" ? "X-Tenant-Id" : "subject"}
          onChange={(e) => onIdentifierValueChange(e.target.value)}
          descriptionPosition="inline"
          description={
            identifierSource === "header" ? (
              "The header whose value becomes the rate limit identifier."
            ) : (
              <>
                Dotted path into the principal JSON (e.g. "subject" or "source.key.meta.org_id").{" "}
                <DocsLink href="https://www.unkey.com/docs/platform/sentinel/principal/overview" />
              </>
            )
          }
        />
      )}
    </div>
  );
}

export function RatelimitPolicySummary() {
  const { control } = useFormContext<RatelimitFormValues>();
  const limit = useWatch({ control, name: "limit" });
  const windowMs = useWatch({ control, name: "windowMs" });
  const identifierSource = useWatch({ control, name: "identifierSource" });
  const identifierValue = useWatch({ control, name: "identifierValue" });

  return (
    <div className="max-w-75 truncate">
      <span className="text-gray-11">
        <Strong>{limit}</Strong> / {windowMs >= 1000 ? `${windowMs / 1000}s` : `${windowMs}ms`}
        <Sep />
        per <Strong>{IDENTIFIER_SOURCE_LABELS[identifierSource]}</Strong>
        {identifierValue && (
          <>
            : <Strong>{identifierValue}</Strong>
          </>
        )}
      </span>
    </div>
  );
}

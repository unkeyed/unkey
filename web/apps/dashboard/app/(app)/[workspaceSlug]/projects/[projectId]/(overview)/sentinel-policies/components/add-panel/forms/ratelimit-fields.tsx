"use client";

import { ChevronDown } from "@unkey/icons";
import {
  FormInput,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@unkey/ui";
import { FormLabel } from "@unkey/ui/src/components/form/form-helpers";
import { useController, useFormContext, useWatch } from "react-hook-form";
import { Sep, Strong } from "./summary-helpers";

// Self-contained ratelimit form types. Not yet wired into the canonical
// PolicyFormValues union — this file is a placeholder for an upcoming
// ratelimit policy variant. Once `policyFormSchema` grows a `ratelimit`
// branch, replace these with `Extract<PolicyFormValues, { type: "ratelimit" }>`
// and the matching key-source enum from ../schema.
export type RateLimitKeySource =
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
  keySource: RateLimitKeySource;
  keyValue: string;
};

const KEY_SOURCE_LABELS: Record<RateLimitKeySource, string> = {
  remoteIp: "IP",
  header: "Header",
  authenticatedSubject: "Subject",
  path: "Path",
  principalField: "Field",
};

const KEY_SOURCE_OPTIONS: { value: RateLimitKeySource; label: string }[] = [
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
    field: { value: keySource, onChange: onKeySourceChange },
  } = useController({ control, name: "keySource" });

  const {
    field: { value: keyValue, onChange: onKeyValueChange },
  } = useController({ control, name: "keyValue" });

  const needsKeyValue = keySource === "header" || keySource === "principalField";

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

      <fieldset className="flex flex-col gap-1.5 border-0 m-0 p-0">
        <FormLabel
          label="Key Source"
          htmlFor="ratelimit-key-source"
          tooltipContent="Determines how the rate limit bucket is keyed (per IP, per header value, per authenticated identity, etc.)."
        />
        <Select
          value={keySource}
          onValueChange={(v) => {
            onKeySourceChange(v as RateLimitKeySource);
            onKeyValueChange("");
          }}
        >
          <SelectTrigger
            id="ratelimit-key-source"
            rightIcon={<ChevronDown className="absolute right-2" iconSize="md-medium" />}
          >
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {KEY_SOURCE_OPTIONS.map((opt) => (
              <SelectItem key={opt.value} value={opt.value}>
                {opt.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </fieldset>

      {needsKeyValue && (
        <FormInput
          label={keySource === "header" ? "Header Name" : "Field Path"}
          value={keyValue}
          placeholder={keySource === "header" ? "X-Tenant-Id" : "subject"}
          onChange={(e) => onKeyValueChange(e.target.value)}
          descriptionPosition="label"
          description={
            keySource === "header"
              ? "The header whose value becomes the rate limit bucket key."
              : 'Dotted path into the principal JSON (e.g. "subject" or "source.key.meta.org_id").'
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
  const keySource = useWatch({ control, name: "keySource" });
  const keyValue = useWatch({ control, name: "keyValue" });

  return (
    <div className="max-w-75 truncate">
      <span className="text-gray-11">
        <Strong>{limit}</Strong> / {windowMs >= 1000 ? `${windowMs / 1000}s` : `${windowMs}ms`}
        <Sep />
        per <Strong>{KEY_SOURCE_LABELS[keySource]}</Strong>
        {keyValue && (
          <>
            : <Strong>{keyValue}</Strong>
          </>
        )}
      </span>
    </div>
  );
}

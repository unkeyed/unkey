"use client";

import { FormInput } from "@unkey/ui";
import { SimpleSelect } from "../simple-select";
import type { RateLimitConfig } from "../types";

const KEY_SOURCE_OPTIONS = [
  { value: "remoteIp", label: "Remote IP" },
  { value: "header", label: "Header" },
  { value: "authenticatedSubject", label: "Authenticated Subject" },
  { value: "path", label: "Request Path" },
  { value: "principalClaim", label: "Principal Claim" },
];

export function RateLimitForm({
  config,
  onChange,
}: {
  config: RateLimitConfig;
  onChange: (config: RateLimitConfig) => void;
}) {
  const needsKeyValue = config.keySource === "header" || config.keySource === "principalClaim";

  return (
    <div className="flex flex-col gap-4">
      <div className="flex gap-3">
        <div className="flex-1">
          <FormInput
            label="Limit"
            type="number"
            value={config.limit}
            onChange={(e) => onChange({ ...config, limit: Number.parseInt(e.target.value) || 0 })}
          />
        </div>
        <div className="flex-1">
          <FormInput
            label="Window (ms)"
            type="number"
            value={config.windowMs}
            onChange={(e) =>
              onChange({ ...config, windowMs: Number.parseInt(e.target.value) || 0 })
            }
            description={config.windowMs > 0 ? `${(config.windowMs / 1000).toFixed(1)}s` : ""}
          />
        </div>
      </div>

      <SimpleSelect
        label="Key Source"
        value={config.keySource}
        options={KEY_SOURCE_OPTIONS}
        onChange={(v) =>
          onChange({
            ...config,
            keySource: v as RateLimitConfig["keySource"],
            keyValue: "",
          })
        }
      />

      {needsKeyValue && (
        <FormInput
          label={config.keySource === "header" ? "Header Name" : "Claim Name"}
          value={config.keyValue}
          placeholder={config.keySource === "header" ? "X-Tenant-Id" : "org_id"}
          onChange={(e) => onChange({ ...config, keyValue: e.target.value })}
        />
      )}
    </div>
  );
}

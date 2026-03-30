"use client";

import { FormInput } from "@unkey/ui";
import type { IPRulesConfig } from "../types";

export function IPRulesForm({
  config,
  onChange,
}: {
  config: IPRulesConfig;
  onChange: (config: IPRulesConfig) => void;
}) {
  return (
    <div className="flex flex-col gap-4">
      <FormInput
        label="Allow CIDRs"
        value={config.allow.join(", ")}
        placeholder="10.0.0.0/8, 192.168.1.0/24"
        description="Comma-separated. Only these IPs are allowed (if non-empty)."
        onChange={(e) =>
          onChange({
            ...config,
            allow: e.target.value
              .split(",")
              .map((s) => s.trim())
              .filter(Boolean),
          })
        }
      />

      <FormInput
        label="Deny CIDRs"
        value={config.deny.join(", ")}
        placeholder="198.51.100.0/24"
        description="Comma-separated. Checked before allow list."
        onChange={(e) =>
          onChange({
            ...config,
            deny: e.target.value
              .split(",")
              .map((s) => s.trim())
              .filter(Boolean),
          })
        }
      />
    </div>
  );
}

"use client";

import { FormInput } from "@unkey/ui";
import type { Control } from "react-hook-form";
import { useController } from "react-hook-form";
import type { PolicyFormValues } from "../schema";

type RatelimitFormValues = Extract<PolicyFormValues, { type: "ratelimit" }>;

export function RateLimitFields({ control }: { control: Control<RatelimitFormValues> }) {
  const {
    field: { value: limit, onChange: onLimitChange },
    fieldState: { error: limitError },
  } = useController({ control, name: "limit" });

  const {
    field: { value: windowMs, onChange: onWindowChange },
    fieldState: { error: windowError },
  } = useController({ control, name: "windowMs" });

  return (
    <div className="flex gap-3">
      <FormInput
        label="Limit"
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
        description={
          windowMs > 0
            ? `${(windowMs / 1000).toFixed(1)}s — time window before the counter resets.`
            : "Time window before the counter resets."
        }
        error={windowError?.message}
      />
    </div>
  );
}

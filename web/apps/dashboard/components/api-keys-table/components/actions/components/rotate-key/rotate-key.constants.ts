import type { FormSelectOption } from "@unkey/ui";

export const GRACE_PERIOD_OPTIONS: FormSelectOption[] = [
  { value: "0", label: "Revoke immediately" },
  { value: "60000", label: "1 minute" },
  { value: "900000", label: "15 minutes" },
  { value: "3600000", label: "1 hour" },
  { value: "21600000", label: "6 hours" },
  { value: "86400000", label: "24 hours" },
];

export const DEFAULT_GRACE_PERIOD = "86400000";

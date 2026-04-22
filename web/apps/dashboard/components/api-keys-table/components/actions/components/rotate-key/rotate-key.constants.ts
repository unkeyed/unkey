import type { FormSelectOption } from "@unkey/ui";

export const GRACE_PERIOD_OPTIONS: FormSelectOption[] = [
  { value: "0", label: "Revoke immediately" },
  { value: "3600000", label: "1 hour" },
  { value: "86400000", label: "24 hours" },
  { value: "604800000", label: "7 days" },
  { value: "2592000000", label: "30 days" },
];

export const DEFAULT_GRACE_PERIOD = "86400000";

import type { Key } from "./schema/keys.schema";

export const KEY_STATUSES = ["enabled", "disabled", "expired"] as const;
export type KeyStatus = (typeof KEY_STATUSES)[number];

export const KEY_STATUS_LABELS: Record<KeyStatus, string> = {
  enabled: "Enabled",
  disabled: "Disabled",
  expired: "Expired",
};

export function keyStatus(key: Key): KeyStatus {
  if (!key.enabled) {
    return "disabled";
  }
  if (key.expires !== null && key.expires < Date.now()) {
    return "expired";
  }
  return "enabled";
}

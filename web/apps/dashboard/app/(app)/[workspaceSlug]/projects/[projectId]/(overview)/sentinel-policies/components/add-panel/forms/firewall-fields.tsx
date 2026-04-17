"use client";

import { Strong } from "./summary-helpers";

/**
 * Firewall has no per-policy configuration in the MVP — every matched
 * request is denied with a fixed 403 / "Forbidden" response. The fields
 * surface exists so the slide-panel layout stays uniform across policy
 * types and so additional configuration can land here without a layout
 * rework.
 */
export function FirewallFields() {
  return (
    <div className="text-gray-11 text-[13px] leading-5">
      Matching requests are denied with HTTP <Strong className="font-mono">403 Forbidden</Strong>.
      Use match conditions below to scope which requests this rule blocks.
    </div>
  );
}

/** Summary row shown next to the collapsed Firewall configuration section. */
export function FirewallPolicySummary() {
  return (
    <div className="max-w-75 truncate">
      <span className="text-gray-11">
        Action: <Strong>Deny</Strong>
      </span>
    </div>
  );
}

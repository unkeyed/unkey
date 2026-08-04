"use client";

import { Strong } from "./summary-helpers";

export function LoggingFields() {
  return (
    <div className="flex flex-col gap-4">
      <div className="text-gray-11 text-[13px] leading-5">
        Records requests handled by the gateway, including request and response bodies, so they
        appear in <Strong>Logs</Strong>. Sensitive headers like{" "}
        <Strong className="font-mono">Authorization</Strong> are redacted automatically. Without an
        enabled logging policy the gateway records nothing.
      </div>
    </div>
  );
}

export function LoggingPolicySummary() {
  return (
    <span className="text-gray-11">
      <Strong>Requests &amp; responses</Strong>
    </span>
  );
}

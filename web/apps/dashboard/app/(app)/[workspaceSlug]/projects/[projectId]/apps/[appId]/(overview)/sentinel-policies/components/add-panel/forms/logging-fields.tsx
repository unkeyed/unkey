"use client";

import { Strong } from "./summary-helpers";

export function LoggingFields() {
  return (
    <div className="flex flex-col gap-4">
      <div className="text-gray-11 text-[13px] leading-5">
        The gateway records each matched request and response, including bodies. The records show in{" "}
        <Strong>Logs</Strong>. A policy without match conditions logs all requests. Add match
        conditions to log only some routes. The gateway redacts sensitive headers such as{" "}
        <Strong className="font-mono">Authorization</Strong>. Without an enabled logging policy, the
        gateway records nothing.
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

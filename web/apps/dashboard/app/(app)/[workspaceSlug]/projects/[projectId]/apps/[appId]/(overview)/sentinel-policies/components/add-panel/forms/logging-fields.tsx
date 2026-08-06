"use client";

import { Switch } from "@/components/ui/switch";
import { useController, useFormContext, useWatch } from "react-hook-form";
import type { PolicyFormValues } from "../schema";
import { Sep, Strong } from "./summary-helpers";

type LoggingFormValues = Extract<PolicyFormValues, { type: "logging" }>;

function CaptureToggle({
  name,
  label,
  description,
}: {
  name: "requestHeaders" | "responseHeaders" | "requestBody" | "responseBody";
  label: string;
  description: string;
}) {
  const { control } = useFormContext<LoggingFormValues>();
  const { field } = useController({ control, name });

  return (
    <div className="flex items-start justify-between gap-4">
      <div className="flex flex-col gap-1">
        <span className="text-[13px] text-gray-12">{label}</span>
        <span className="text-[12px] text-gray-10">{description}</span>
      </div>
      <Switch size="sm" checked={field.value} onCheckedChange={field.onChange} aria-label={label} />
    </div>
  );
}

export function LoggingFields() {
  return (
    <div className="flex flex-col gap-4">
      <div className="text-gray-11 text-[13px] leading-5">
        The gateway always logs the method, host, path, status, and latency of each request. This
        policy adds more data for matched requests. If you set no match conditions, the policy
        applies to all requests. The gateway always redacts sensitive headers such as{" "}
        <Strong className="font-mono">Authorization</Strong>.
      </div>
      <CaptureToggle
        name="requestHeaders"
        label="Request headers"
        description="Also stores query parameters, the user agent, and the client IP."
      />
      <CaptureToggle
        name="responseHeaders"
        label="Response headers"
        description="Store response headers."
      />
      <CaptureToggle
        name="requestBody"
        label="Request body"
        description="Store the request body."
      />
      <CaptureToggle
        name="responseBody"
        label="Response body"
        description="Store the response body."
      />
    </div>
  );
}

export function LoggingPolicySummary() {
  const { control } = useFormContext<LoggingFormValues>();
  const requestHeaders = useWatch({ control, name: "requestHeaders" });
  const responseHeaders = useWatch({ control, name: "responseHeaders" });
  const requestBody = useWatch({ control, name: "requestBody" });
  const responseBody = useWatch({ control, name: "responseBody" });

  const parts = [
    requestHeaders && "req headers",
    responseHeaders && "res headers",
    requestBody && "req body",
    responseBody && "res body",
  ].filter(Boolean);

  return (
    <span className="text-gray-11">
      <Strong>Basic log</Strong>
      {parts.length > 0 && (
        <>
          <Sep />+ <Strong>{parts.join(" & ")}</Strong>
        </>
      )}
    </span>
  );
}

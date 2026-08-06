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
  name: "headers" | "bodies";
  label: string;
  description: string;
}) {
  const { control } = useFormContext<LoggingFormValues>();
  const { field } = useController({ control, name });

  return (
    <div className="flex items-start gap-3">
      <Switch size="sm" checked={field.value} onCheckedChange={field.onChange} aria-label={label} />
      <div className="flex flex-col gap-0.5">
        <span className="text-gray-12 text-[13px] leading-5">{label}</span>
        <span className="text-gray-11 text-[12px] leading-4">{description}</span>
      </div>
    </div>
  );
}

export function LoggingFields() {
  return (
    <div className="flex flex-col gap-4">
      <div className="text-gray-11 text-[13px] leading-5">
        The gateway always records a basic log entry for each request: method, host, path, status,
        and latency. You cannot turn this off. This policy adds sensitive data to the entries of
        matched requests. A policy without match conditions applies to all requests. The gateway
        redacts sensitive headers such as <Strong className="font-mono">Authorization</Strong>{" "}
        before it stores them.
      </div>
      <CaptureToggle
        name="headers"
        label="Headers"
        description="Store request and response headers, the query string, and query parameters."
      />
      <CaptureToggle
        name="bodies"
        label="Bodies"
        description="Store request and response bodies."
      />
    </div>
  );
}

export function LoggingPolicySummary() {
  const { control } = useFormContext<LoggingFormValues>();
  const headers = useWatch({ control, name: "headers" });
  const bodies = useWatch({ control, name: "bodies" });

  const parts = [headers && "headers", bodies && "bodies"].filter(Boolean);

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

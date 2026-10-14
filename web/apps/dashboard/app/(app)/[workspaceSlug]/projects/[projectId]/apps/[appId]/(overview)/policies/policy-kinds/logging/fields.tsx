"use client";

import { Switch } from "@/components/ui/switch";
import { IconCircleInfoOutline18 } from "@unkey/icons";
import { AlertBanner, AlertBannerDescription, InfoTooltip } from "@unkey/ui";
import { useController, useFormContext } from "react-hook-form";
import { LOGGING_FIELDS, type LoggingFieldName, type LoggingFormValues } from "./model";

function CaptureToggle({
  name,
  label,
  hint,
}: {
  name: LoggingFieldName;
  label: string;
  hint?: string;
}) {
  const { control } = useFormContext<LoggingFormValues>();
  const { field } = useController({ control, name });

  return (
    <div className="flex items-center justify-between gap-4">
      <span className="flex items-center gap-1.5 text-sm text-gray-12">
        {label}
        {hint ? (
          <InfoTooltip content={hint} triggerClassName="flex text-gray-9 hover:text-gray-11">
            <IconCircleInfoOutline18 className="size-3.5" />
          </InfoTooltip>
        ) : null}
      </span>
      <Switch
        size="sm"
        checked={field.value}
        onCheckedChange={field.onChange}
        aria-label={label}
        className="shrink-0"
      />
    </div>
  );
}

export function LoggingFields() {
  return (
    <div className="flex flex-col gap-4">
      <AlertBanner className="bg-grayA-2">
        <IconCircleInfoOutline18 className="size-4 text-gray-11" />
        <AlertBannerDescription className="text-xs">
          Unkey always logs the method, host, path, status and latency. It always redacts sensitive
          headers, such as Authorization.
        </AlertBannerDescription>
      </AlertBanner>
      {LOGGING_FIELDS.map(({ name, label, hint }) => (
        <CaptureToggle key={name} name={name} label={label} hint={hint} />
      ))}
    </div>
  );
}

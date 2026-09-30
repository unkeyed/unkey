"use client";

import { Switch } from "@/components/ui/switch";
import { InfoTooltip, Loading } from "@unkey/ui";
import type React from "react";
import { useId } from "react";

type SettingsToggleRowProps = {
  title: React.ReactNode;
  description?: React.ReactNode;
  checked: boolean;
  onCheckedChange: (checked: boolean) => void;
  pending?: boolean;
  disabledReason?: string;
};

export function SettingsToggleRow({
  title,
  description,
  checked,
  onCheckedChange,
  pending = false,
  disabledReason,
}: SettingsToggleRowProps) {
  const titleId = useId();
  const descriptionId = useId();

  return (
    <div className="flex items-center justify-between gap-4 px-5 py-5">
      <div className="flex min-w-0 flex-col gap-1">
        <div id={titleId} className="text-sm font-medium text-gray-12">
          {title}
        </div>
        {description ? (
          <p id={descriptionId} className="text-xs leading-5 text-gray-11">
            {description}
          </p>
        ) : null}
      </div>
      <div className="flex shrink-0 items-center gap-2">
        {pending ? <Loading size={14} className="text-gray-11" /> : null}
        <InfoTooltip
          content={disabledReason}
          disabled={!disabledReason}
          position={{ side: "top" }}
          asChild
        >
          <span className="inline-flex">
            <Switch
              checked={checked}
              onCheckedChange={onCheckedChange}
              disabled={pending || Boolean(disabledReason)}
              aria-labelledby={titleId}
              aria-describedby={description ? descriptionId : undefined}
            />
          </span>
        </InfoTooltip>
      </div>
    </div>
  );
}

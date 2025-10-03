"use client";
import { Switch } from "@/components/ui/switch";

import { CircleInfo } from "@unkey/icons";
import { InfoTooltip } from "@unkey/ui";
import { useState } from "react";
import type { ReactNode } from "react";

type ExpandableSettingsProps = {
  icon: ReactNode;
  title: string;
  description?: string;
  children: ReactNode | ((enabled: boolean) => ReactNode);
  defaultChecked?: boolean;
  onCheckedChange?: (checked: boolean) => void;
  disabled?: boolean;
  disabledTooltip?: string;
};

export const ExpandableSettings = ({
  icon,
  title,
  description,
  children,
  defaultChecked = false,
  onCheckedChange,
  disabled = false,
  disabledTooltip = "You need to have a valid API name",
}: ExpandableSettingsProps) => {
  const [isEnabled, setIsEnabled] = useState(defaultChecked);

  const handleCheckedChange = (checked: boolean) => {
    if (disabled) {
      return;
    }
    setIsEnabled(checked);
    onCheckedChange?.(checked);
  };
  const handleSwitchClick = (e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();
  };

  const handleHeaderClick = (e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();
    handleCheckedChange(!isEnabled);
  };

  return (
    <InfoTooltip content={disabledTooltip} disabled={!disabled} asChild>
      <div className={disabled ? "opacity-50 pointer-events-none" : ""}>
        {/* Header */}
        <button
          type="button"
          className="flex items-center border rounded-lg border-grayA-3 py-1 pl-[14px] pr-3 cursor-pointer w-full"
          onClick={handleHeaderClick}
        >
          <div className="flex items-center">
            {icon}
            <div className="ml-3 mr-2 text-gray-12 font-medium text-[13px] leading-6">{title}</div>
            {description && (
              <InfoTooltip content={description}>
                <CircleInfo className="text-gray-8 flex-shrink-0" iconsize="sm-regular" />
              </InfoTooltip>
            )}
          </div>
          {/* biome-ignore lint/a11y/useKeyWithClickEvents: no need */}
          <div onClick={handleSwitchClick} className="ml-auto">
            <Switch
              checked={isEnabled}
              onCheckedChange={handleCheckedChange}
              disabled={disabled}
              className="
              ml-auto
              h-4 w-8
              data-[state=checked]:bg-success-9
              data-[state=checked]:ring-2
              data-[state=checked]:ring-successA-5
              data-[state=unchecked]:bg-gray-3
              data-[state=unchecked]:ring-2
              data-[state=unchecked]:ring-grayA-3
              [&>span]:h-3.5 [&>span]:w-3.5
            "
              thumbClassName="h-[14px] w-[14px] data-[state=unchecked]:bg-grayA-9 data-[state=checked]:bg-white"
            />
          </div>
        </button>
        {/* Expandable Content */}
        {isEnabled && !disabled && (
          <div className="relative -mb-3">
            <div
              className="absolute top-0 bottom-0 w-px bg-grayA-4"
              style={{ left: `${14 + 4}px` }}
            />
            {/* Content */}
            <div className="py-6 px-10 text-start">
              {typeof children === "function" ? children(isEnabled) : children}
            </div>
          </div>
        )}
      </div>
    </InfoTooltip>
  );
};

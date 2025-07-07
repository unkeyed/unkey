import { Switch } from "@/components/ui/switch";
import { CircleInfo } from "@unkey/icons";
import { useState } from "react";
import type { ReactNode } from "react";

type ExpandableSettingsProps = {
  icon: ReactNode;
  title: string;
  children: ReactNode;
  defaultChecked?: boolean;
  onCheckedChange?: (checked: boolean) => void;
};

export const ExpandableSettings = ({
  icon,
  title,
  children,
  defaultChecked = false,
  onCheckedChange,
}: ExpandableSettingsProps) => {
  const [isEnabled, setIsEnabled] = useState(defaultChecked);

  const handleCheckedChange = (checked: boolean) => {
    setIsEnabled(checked);
    onCheckedChange?.(checked);
  };

  return (
    <div>
      {/* Header */}
      <div className="flex items-center border rounded-lg border-grayA-3 py-1 pl-[14px] pr-3">
        <div className="flex items-center">
          {icon}
          <div className="ml-3 mr-2 text-gray-12 font-medium text-[13px] leading-6">{title}</div>
          <CircleInfo className="text-gray-8 flex-shrink-0" size="sm-regular" />
        </div>
        <Switch
          checked={isEnabled}
          onCheckedChange={handleCheckedChange}
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

      {/* Expandable Content */}
      {isEnabled && (
        <div className="relative -mb-3">
          <div
            className="absolute top-0 bottom-0 w-px bg-grayA-4"
            style={{ left: `${14 + 4}px` }}
          />
          {/* Content */}
          <div className="py-6 px-10">{children}</div>
        </div>
      )}
    </div>
  );
};

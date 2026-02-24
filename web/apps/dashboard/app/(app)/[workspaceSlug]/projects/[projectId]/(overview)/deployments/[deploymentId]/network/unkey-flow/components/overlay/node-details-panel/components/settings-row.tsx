import { CircleQuestion } from "@unkey/icons";
import type { ReactNode } from "react";

type Setting = {
  label: string;
  value: string | number | ReactNode;
  icon?: ReactNode;
};

type SettingsSectionProps = {
  title: string;
  settings: Setting[];
};

export const SettingsSection = ({ title, settings }: SettingsSectionProps) => {
  return (
    <div className="flex px-4 w-full mt-4 flex-col">
      <div className="flex items-center gap-3 w-full">
        <div className="text-gray-9 text-xs whitespace-nowrap">{title}</div>
        <div className="h-0.5 bg-grayA-3 rounded-xs flex-1 min-w-[115px]" />
      </div>
      <div className="mt-5" />
      <div className="flex flex-col gap-3">
        {settings.map(({ label, value, icon }) => (
          <div key={label} className="grid grid-cols-[auto_140px_1fr] items-center gap-3">
            {icon ?? (
              <CircleQuestion className="size-[14px] text-gray-12 shrink-0" iconSize="md-regular" />
            )}
            <span className="text-gray-11 text-xs">{label}</span>
            <div className="text-gray-12 font-medium text-[13px]">
              {typeof value === "string" || typeof value === "number" ? value : value}
            </div>
          </div>
        ))}
      </div>
    </div>
  );
};

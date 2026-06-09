"use client";
import { Switch } from "@/components/ui/switch";
import { forwardRef } from "react";

type FeatureCardProps = {
  icon: React.ReactNode;
  title: string;
  description: string;
  checked: boolean;
  onCheckedChange: (checked: boolean) => void;
  switchProps?: React.ComponentPropsWithoutRef<typeof Switch>;
};

export const ProtectionSwitch = forwardRef<HTMLDivElement, FeatureCardProps>(
  ({ icon, title, description, checked, onCheckedChange, switchProps, ...rest }, ref) => {
    return (
      <div
        ref={ref}
        className="flex flex-row py-5 pl-5 pr-[26px] gap-14 justify-between border rounded-xl border-grayA-5 bg-white dark:bg-black items-center"
        {...rest}
      >
        <div className="flex flex-col gap-4">
          <div className="flex gap-3">
            <div className="p-1.5 bg-grayA-3 rounded-md border border-grayA-3">{icon}</div>
            <div className="text-sm font-medium text-gray-12">{title}</div>
          </div>
          <div className="text-gray-9 text-xs">{description}</div>
        </div>
        <Switch checked={checked} onCheckedChange={onCheckedChange} {...switchProps} />
      </div>
    );
  },
);

ProtectionSwitch.displayName = "ProtectionSwitch";

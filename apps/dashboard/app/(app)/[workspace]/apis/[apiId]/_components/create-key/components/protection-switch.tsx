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
        <Switch
          className="
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
          checked={checked}
          onCheckedChange={onCheckedChange}
          {...switchProps}
        />
      </div>
    );
  },
);

ProtectionSwitch.displayName = "ProtectionSwitch";

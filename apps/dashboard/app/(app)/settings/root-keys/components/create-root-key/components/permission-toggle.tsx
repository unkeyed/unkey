"use client";
import React from "react";
import type { CheckedState } from "@radix-ui/react-checkbox";
import { ChevronRight } from "@unkey/icons";
import { Checkbox, InfoTooltip } from "@unkey/ui";

type PermissionToggleProps = {
  category: string | React.ReactNode;
  checked: CheckedState;
  setChecked: (checked: boolean) => void;
  label: string | React.ReactNode;
  description: string;
  collapsible?: boolean;
};

export const PermissionToggle: React.FC<PermissionToggleProps> = ({
  category,
  checked,
  setChecked,
  label,
  description,
}) => {
  // Convert label to string for tooltip if it's a React node
  const labelString = typeof label === 'string' ? label : String(label);

  return (
    <div className="flex flex-row items-center justify-evenly gap-4 transition-all pl-3 h-full my-1 ml-2 w-full hover:bg-grayA-3 rounded-lg">
      <Checkbox
        size="lg"
        checked={checked}
        onCheckedChange={(checked) => {
          if (checked === "indeterminate") {
            setChecked(false);
          } else {
            setChecked(!checked);
          }
        }}
      />
      <div className="flex flex-col text-left min-w-48 w-full ml-2">
        <div className="inline-flex items-center gap-2">
          <span className="text-sm w-fit">{category}</span>
          <ChevronRight size="sm-regular" className="text-grayA-8" />
          <span className="text-sm w-full">{label}</span>
        </div>
        <InfoTooltip content={description} className="w-full text-left">
          <p className="text-xs text-gray-10 text-left max-w-[235px] truncate mt-1">
            {description}
          </p>
        </InfoTooltip>
      </div>
    </div>
  );
};

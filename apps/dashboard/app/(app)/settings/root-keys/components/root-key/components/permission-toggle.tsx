"use client";
import type { CheckedState } from "@radix-ui/react-checkbox";
import { ChevronRight } from "@unkey/icons";
import { Checkbox, InfoTooltip } from "@unkey/ui";
import type React from "react";

type PermissionToggleProps = {
  category: string | React.ReactNode;
  checked: CheckedState;
  setChecked: (checked: boolean) => void;
  label: string | React.ReactNode;
  description: string;
};

export const PermissionToggle: React.FC<PermissionToggleProps> = ({
  category,
  checked,
  setChecked,
  label,
  description,
}) => {
  return (
    <div className="flex flex-row items-center justify-start gap-4 transition-all pl-3 h-full mb-1 ml-2 w-full hover:bg-grayA-3 rounded-lg">
      <Checkbox
        size="lg"
        checked={checked}
        onCheckedChange={(next) => {
          // Treat indeterminate as unchecked; otherwise set to the boolean next value
          setChecked(next === true);
        }}
      />
      <div className="flex flex-col text-left min-w-48 max-w-full mr-2 gap-1">
        <div className="inline-flex items-center gap-2 w-full">
          <span className="text-sm w-fit">{category}</span>
          <ChevronRight size="sm-regular" className="text-grayA-8" />
          {<span className="text-sm w-full">{label}</span>}
        </div>
        <InfoTooltip content={description} className="w-full text-left">
          <p className="text-xs text-gray-10 text-left max-w-[245px] w-full truncate mr-2">
            {description}
          </p>
        </InfoTooltip>
      </div>
    </div>
  );
};

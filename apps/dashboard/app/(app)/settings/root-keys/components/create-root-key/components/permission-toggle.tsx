"use client";
import type { CheckedState } from "@radix-ui/react-checkbox";
import { Checkbox } from "@unkey/ui";

type PermissionToggleProps = {
  checked: CheckedState;
  setChecked: (checked: boolean) => void;
  label: string | React.ReactNode;
  description: string;
  collapsible?: boolean;
};

export const PermissionToggle: React.FC<PermissionToggleProps> = ({
  checked,
  setChecked,
  label,
  description,
}) => {
  return (
    <div className="flex flex-row items-center justify-evenly gap-3 transition-all px-4 h-full py-0 my-0">
      <Checkbox
        checked={checked}
        onCheckedChange={(checked) => {
          if (checked === "indeterminate") {
            setChecked(false);
          } else {
            setChecked(!checked);
          }
        }}
      />
      <div className="flex flex-col text-left min-w-48 w-full">
        <p className="text-sm w-full">{label}</p>
        <p className="text-xs text-content-subtle w-full">{description}</p>
      </div>
    </div>
  );
};

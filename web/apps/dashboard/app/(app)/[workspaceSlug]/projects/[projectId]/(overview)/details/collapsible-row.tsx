import { ChevronDown } from "@unkey/icons";
import { Button } from "@unkey/ui";
import type { ReactNode } from "react";

type CollapsibleRowProps = {
  icon: ReactNode;
  title: string;
  onToggle?: () => void;
};
export function CollapsibleRow({ icon, title, onToggle }: CollapsibleRowProps) {
  return (
    <div className="border border-gray-4 border-t-0 first:border-t first:rounded-t-[14px] last:rounded-b-[14px] w-full px-4 py-3 flex justify-between items-center">
      <div className="flex items-center">
        {icon}
        <div className="text-gray-12 font-medium text-xs ml-3 mr-2">{title}</div>
      </div>
      <Button size="icon" variant="ghost" onClick={onToggle}>
        <ChevronDown className="text-grayA-9 !size-3" />
      </Button>
    </div>
  );
}

"use client";

import { CollapsibleTrigger } from "@/components/ui/collapsible";
import { cn } from "@/lib/utils";
import type { CheckedState } from "@radix-ui/react-checkbox";
import { ChevronDown } from "@unkey/icons";
import { Checkbox } from "@unkey/ui";

type ExpandableCategoryProps = {
  category: string;
  description: string;
  checked: CheckedState | undefined;
  setChecked: () => void;
} & React.ComponentProps<typeof CollapsibleTrigger>;

export const ExpandableCategory = ({
  category,
  description,
  checked,
  setChecked,
  ...props
}: ExpandableCategoryProps) => {
  return (
    <div className="flex flex-row items-center justify-evenly gap-3 transition-all px-4 h-full py-0 my-0">
      <div className="flex items-center justify-center">
        <Checkbox checked={checked} onCheckedChange={() => setChecked()} />
      </div>
      <CollapsibleTrigger
        {...props}
        className={cn(
          "flex items-center justify-evenly gap-3 transition-all pb-2 px-4 [&[data-state=open]>svg]:rotate-180 w-full",
        )}
      >
        <div className="flex flex-col text-left min-w-48 w-full">
          <p className="text-sm w-full">{category}</p>
          <p className="text-xs text-content-subtle w-full">{description}</p>
        </div>
        <ChevronDown className="w-4 h-4 transition-transform duration-200 ml-auto" />
      </CollapsibleTrigger>
    </div>
  );
};

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
  setChecked: (checked: CheckedState) => void;
  count: number;
} & React.ComponentProps<typeof CollapsibleTrigger>;

export const ExpandableCategory = ({
  category,
  description,
  checked,
  setChecked,
  count,
  ...props
}: ExpandableCategoryProps) => {
  if (count === 0) {
    return null;
  }
  return (
    <div className="flex flex-row items-center justify-evenly gap-3 transition-all pl-3 pr-2 h-full my-2">
      <div className="flex items-center justify-center">
        <Checkbox
          checked={checked}
          onCheckedChange={(next) => setChecked(next)}
          size="lg"
          aria-label={`Toggle ${category} permissions`}
        />
      </div>
      <CollapsibleTrigger
        {...props}
        className={cn(
          "flex items-center justify-evenly gap-3 transition-all pl-2 pr-2.5 [&[data-state=open]>svg]:rotate-180 w-full",
          props.className,
        )}
      >
        <div className="flex flex-col text-left min-w-48 w-full">
          <p className="text-sm w-full">{category}</p>
          <p className="text-xs text-gray-10 w-full truncate">{description}</p>
        </div>
        <ChevronDown className="w-4 h-4 transition-transform duration-200 ml-auto text-grayA-8" />
      </CollapsibleTrigger>
    </div>
  );
};

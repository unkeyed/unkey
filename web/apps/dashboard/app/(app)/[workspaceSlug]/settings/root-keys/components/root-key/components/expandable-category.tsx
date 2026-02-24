"use client";

import { CollapsibleTrigger } from "@/components/ui/collapsible";
import { cn } from "@/lib/utils";
import type { CheckedState } from "@radix-ui/react-checkbox";
import { CaretRight } from "@unkey/icons";
import { Checkbox } from "@unkey/ui";
import { type ComponentPropsWithoutRef, type ElementRef, forwardRef } from "react";

export type ExpandableCategoryProps = {
  category: string;
  description?: string;
  checked: CheckedState | undefined;
  setChecked: (checked: CheckedState) => void;
  count: number;
} & Omit<ComponentPropsWithoutRef<typeof CollapsibleTrigger>, "children" | "asChild">;

const ExpandableCategory = forwardRef<
  ElementRef<typeof CollapsibleTrigger>,
  ExpandableCategoryProps
>(({ category, description, checked, setChecked, count, ...props }, ref) => {
  if (count === 0) {
    return null;
  }
  return (
    <div className="flex flex-row items-center justify-start gap-3 pl-3 h-full my-2">
      <div className="flex items-center justify-center">
        <Checkbox
          checked={checked}
          onCheckedChange={(next) => setChecked(next)}
          size="lg"
          aria-label={`Toggle ${category} permissions`}
        />
      </div>
      <CollapsibleTrigger
        ref={ref}
        {...props}
        className={cn(
          "flex items-center justify-start gap-3 pl-2 pr-2.5 [&[data-state=open]>svg]:rotate-90 w-full",
          props.className,
        )}
      >
        <div className="flex flex-col text-left min-w-48 w-full">
          <p className="text-sm w-full">{category}</p>
          {description ? (
            <p className="text-xs text-gray-10 w-full truncate">{description}</p>
          ) : null}
        </div>
        <CaretRight
          className="w-4 h-4 transition-transform duration-200 ml-auto text-grayA-7"
          aria-hidden="true"
        />
      </CollapsibleTrigger>
    </div>
  );
});

ExpandableCategory.displayName = "ExpandableCategory";

export { ExpandableCategory };

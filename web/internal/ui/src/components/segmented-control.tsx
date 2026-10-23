import { Radio as RadioPrimitive } from "@base-ui/react/radio";
import { RadioGroup as RadioGroupPrimitive } from "@base-ui/react/radio-group";
// biome-ignore lint/correctness/noUnusedImports: Required by this package's classic JSX runtime.
import * as React from "react";
import { cn } from "../lib/utils";

function SegmentedControl<Value>({ className, ...props }: RadioGroupPrimitive.Props<Value>) {
  return (
    <RadioGroupPrimitive
      className={(state) =>
        cn(
          "inline-flex gap-1 rounded-lg bg-grayA-3 p-1",
          typeof className === "function" ? className(state) : className,
        )
      }
      {...props}
    />
  );
}

function SegmentedControlItem<Value>({ className, ...props }: RadioPrimitive.Root.Props<Value>) {
  return (
    <RadioPrimitive.Root
      className={(state) =>
        cn(
          "flex min-h-8 min-w-0 flex-1 cursor-pointer items-center justify-center rounded-md border border-transparent px-3 text-sm text-gray-11 transition-colors hover:text-gray-12 focus:outline-hidden focus-visible:ring-2 focus-visible:ring-gray-8 focus-visible:ring-offset-2 focus-visible:ring-offset-background data-checked:border-grayA-5 data-checked:bg-raised data-checked:text-gray-12 data-checked:shadow-xs data-disabled:pointer-events-none data-disabled:opacity-50",
          typeof className === "function" ? className(state) : className,
        )
      }
      {...props}
    />
  );
}

export { SegmentedControl, SegmentedControlItem };

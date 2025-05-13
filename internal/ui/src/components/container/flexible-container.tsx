// biome-ignore lint: React in this context is used throughout, so biome will change to types because no APIs are used even though React is needed.
import * as React from "react";
import { cn } from "../../lib/utils";
import { type VariantProps, cva } from "class-variance-authority";

const flexibleContainerVariants = cva(
  "flex flex-col w-full h-full", // Base padding and centering
  {
    variants: {
      padding: {
        none: "p-0 m-0",
        small: "px-4 py-2",
        medium: "px-6 py-4",
        large: "px-8 py-6",
      },
      align: {
        center: "items-center",
        start: "items-start",
        end: "items-end",
      },
      justify: {
        center: "justify-center",
        start: "justify-start",
        end: "justify-end",
      },
    },
    defaultVariants: {
      align: "center",
      justify: "center",
      padding: "none",
    },
  },
);
type FlexibleContainerProps = React.HTMLAttributes<HTMLDivElement> &
  VariantProps<typeof flexibleContainerVariants>;

export const FlexibleContainer = ({
  className,
  children,
  align,
  justify,
  padding,
}: FlexibleContainerProps) => {
  return (
    <div className={cn(flexibleContainerVariants({ align, justify, padding, className }))}>
      {children}
    </div>
  );
};

FlexibleContainer.displayName = "FlexibleContainer";

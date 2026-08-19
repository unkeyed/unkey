"use client";

import { type VariantProps, cva } from "class-variance-authority";
import * as React from "react";

import { cn } from "@/lib/utils";

const labelVariants = cva(
  "text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70",
);

const Label = React.forwardRef<
  React.ElementRef<"label">,
  React.ComponentPropsWithoutRef<"label"> & VariantProps<typeof labelVariants>
>(({ className, ...props }, ref) => (
  // biome-ignore lint/a11y/noLabelWithoutControl: reusable Label primitive; consumers supply htmlFor / the associated control
  <label ref={ref} className={cn(labelVariants(), className)} {...props} />
));
Label.displayName = "Label";

export { Label };

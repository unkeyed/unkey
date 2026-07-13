"use client";

import * as React from "react";

import { cn } from "../lib/utils";

// Base UI has no VisuallyHidden primitive; the idiomatic replacement is the
// `sr-only` utility on a plain span.
const VisuallyHidden = React.forwardRef<HTMLSpanElement, React.ComponentPropsWithoutRef<"span">>(
  ({ className, ...props }, ref) => (
    <span ref={ref} className={cn("sr-only", className)} {...props} />
  ),
);
VisuallyHidden.displayName = "VisuallyHidden";

export { VisuallyHidden };

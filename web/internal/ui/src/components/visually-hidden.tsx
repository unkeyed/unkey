"use client";

import type * as React from "react";

import { cn } from "../lib/utils";

// Base UI has no VisuallyHidden primitive; the idiomatic replacement is the
// `sr-only` utility on a plain span.
function VisuallyHidden({
  className,
  ref,
  ...props
}: React.ComponentPropsWithoutRef<"span"> & { ref?: React.Ref<HTMLSpanElement> }) {
  return <span ref={ref} className={cn("sr-only", className)} {...props} />;
}

export { VisuallyHidden };

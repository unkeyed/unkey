import * as React from "react";
import { cn } from "../../lib/utils";

/**
 * Constrains and centers a page's content, applying the standard horizontal
 * gutters. Wrap the {@link PageHeader} and the page body in a single shell so
 * they share the same measure.
 */
const PageShell = React.forwardRef<HTMLDivElement, React.HTMLAttributes<HTMLDivElement>>(
  ({ className, ...props }, ref) => (
    <div
      ref={ref}
      className={cn("w-full max-w-7xl mx-auto px-4 lg:px-6 xl:px-10 pt-2", className)}
      {...props}
    />
  ),
);
PageShell.displayName = "PageShell";

export { PageShell };

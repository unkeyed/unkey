import * as React from "react";
import { cn } from "../../lib/utils";

/** Constrains and centers a page's content with the standard horizontal gutters. */
const PageBody = React.forwardRef<HTMLDivElement, React.HTMLAttributes<HTMLDivElement>>(
  ({ className, ...props }, ref) => (
    <div
      ref={ref}
      className={cn(
        "mx-auto w-full max-w-7xl px-4",
        "group-data-[width=full]/page:max-w-none group-data-[width=full]/page:px-0",
        className,
      )}
      {...props}
    />
  ),
);
PageBody.displayName = "PageBody";

export { PageBody };

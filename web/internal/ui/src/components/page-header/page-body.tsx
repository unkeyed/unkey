import type * as React from "react";
import { cn } from "../../lib/utils";

/** Constrains and centers a page's content with the standard horizontal gutters. */
function PageBody({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div
      className={cn(
        "mx-auto w-full max-w-7xl px-4",
        "group-data-[width=full]/page:max-w-none group-data-[width=full]/page:px-0",
        className,
      )}
      {...props}
    />
  );
}

export { PageBody };

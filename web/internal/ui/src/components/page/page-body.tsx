import type * as React from "react";
import { cn } from "../../lib/utils";

/**
 * Constrains and centers a page's content with the standard gutters and
 * vertical rhythm. The full-width variant drops all of it so dense views
 * (log tables) run edge to edge.
 */
function PageBody({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div
      className={cn(
        "mx-auto flex w-full max-w-7xl flex-col gap-5 px-6 pt-6 pb-20",
        "group-data-[width=full]/page:max-w-none group-data-[width=full]/page:gap-0 group-data-[width=full]/page:p-0",
        className,
      )}
      {...props}
    />
  );
}

export { PageBody };

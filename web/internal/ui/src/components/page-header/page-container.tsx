import * as React from "react";
import { cn } from "../../lib/utils";

type PageWidth = "default" | "full";

/**
 * The page frame. Compose a {@link PageHeader} and {@link PageBody} inside it;
 * `width` flows to them via a `group/page` data attribute so they restyle
 * themselves — `full` runs the body edge to edge while the header keeps its
 * gutter and divider.
 */
const PageContainer = React.forwardRef<
  HTMLDivElement,
  React.HTMLAttributes<HTMLDivElement> & { width?: PageWidth }
>(({ width = "default", className, ...props }, ref) => (
  <div
    ref={ref}
    data-width={width}
    className={cn("group/page flex w-full flex-col", className)}
    {...props}
  />
));
PageContainer.displayName = "PageContainer";

export { PageContainer };

import type * as React from "react";
import { cn } from "../../lib/utils";

type PageWidth = "default" | "full";

/**
 * The page frame. Compose a {@link PageHeader} and {@link PageBody} inside it;
 * `width` flows to them via a `group/page` data attribute so they restyle
 * themselves — `full` runs the body edge to edge while the header keeps its
 * gutter and divider.
 */
function PageContainer({
  width = "default",
  className,
  ...props
}: React.ComponentProps<"div"> & { width?: PageWidth }) {
  return (
    <div
      data-width={width}
      className={cn("group/page flex w-full flex-col", className)}
      {...props}
    />
  );
}

export { PageContainer };

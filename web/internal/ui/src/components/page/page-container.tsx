import type * as React from "react";
import { cn } from "../../lib/utils";
import { STICKY_ROOT_CLASS_NAME } from "./sticky-stack";

type PageWidth = "default" | "full";

/**
 * The page frame. Compose a {@link PageHeader} and {@link PageBody} inside it;
 * `width` flows to them via a `group/page` data attribute so they restyle
 * themselves — `full` runs the body edge to edge while the header keeps its
 * gutter and divider. A `full` frame is also the sticky root: chrome that
 * sits in a `StickySlot` pins under the top nav.
 */
function PageContainer({
  width = "default",
  className,
  ...props
}: React.ComponentProps<"div"> & { width?: PageWidth }) {
  return (
    <div
      data-width={width}
      className={cn(
        "group/page flex w-full flex-col md:data-[width=full]:flex-1",
        STICKY_ROOT_CLASS_NAME,
        className,
      )}
      {...props}
    />
  );
}

export { PageContainer };

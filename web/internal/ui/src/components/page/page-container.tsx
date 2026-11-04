"use client";
import { type ComponentProps, createContext, use, useState } from "react";
import { cn } from "../../lib/utils";
import { ControlsSlotProvider } from "./controls-slots";
import { useWarnIfMissing } from "./dev-warn";
import { STICKY_ROOT_CLASS_NAME } from "./sticky-stack";

type PageWidth = "default" | "full";

const PageElementContext = createContext<HTMLElement | null>(null);

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
  children,
  ...props
}: Omit<ComponentProps<"div">, "ref"> & { width?: PageWidth }) {
  const [page, setPage] = useState<HTMLElement | null>(null);
  return (
    <div
      {...props}
      ref={setPage}
      data-width={width}
      className={cn(
        "group/page flex w-full flex-col md:data-[width=full]:flex-1",
        STICKY_ROOT_CLASS_NAME,
        className,
      )}
    >
      <PageElementContext.Provider value={page}>
        <ControlsSlotProvider>{children}</ControlsSlotProvider>
      </PageElementContext.Provider>
    </div>
  );
}

function usePageElement(consumer: string) {
  const page = use(PageElementContext);
  useWarnIfMissing(
    page !== null,
    `${consumer} is not inside a PageContainer, so it centers on the viewport.`,
  );
  return page;
}

export { PageContainer, usePageElement };

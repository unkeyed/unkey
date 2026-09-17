"use client";

import { ResizablePanel } from "@/components/logs/details/resizable-panel";
import { IconXmarkOutline18 } from "@unkey/icons";
import { Button, Skeleton } from "@unkey/ui";
import { DEFAULT_DRAGGABLE_WIDTH, createPanelStyle } from "..";

type Props = {
  distanceToTop: number;
  onClose: () => void;
};

const SECTION_BODY_HEIGHTS = ["h-20", "h-14", "h-20", "h-14"];
const FOOTER_LABEL_WIDTHS = ["w-10", "w-12", "w-24", "w-20", "w-32"];

/**
 * Placeholder panel shown while a request log is being fetched.
 *
 * The drawer previously rendered nothing until the log resolved, so clicking a
 * row produced no visible change and people clicked again. Mirrors the real
 * panel's header, sections, and footer so swapping in the loaded log does not
 * shift the layout.
 */
export const LogDetailsSkeleton = ({ distanceToTop, onClose }: Props) => {
  return (
    <ResizablePanel
      onClose={onClose}
      className="bg-gray-1 font-mono drop-shadow-2xl z-20 absolute right-0 overflow-y-auto"
      style={{
        ...createPanelStyle(distanceToTop),
        width: `${DEFAULT_DRAGGABLE_WIDTH}px`,
      }}
    >
      <div role="status" aria-live="polite" aria-busy="true">
        <span className="sr-only">Loading request details</span>

        <div className="border-b flex justify-between items-center border-gray-4 h-[50px] px-4 py-2">
          <div className="flex gap-2 items-center min-w-0 flex-1">
            <Skeleton className="h-5 w-12 rounded-md" />
            <Skeleton className="h-3 flex-1 max-w-[200px]" />
            <Skeleton className="h-5 w-10 rounded-md" />
          </div>
          <Button size="icon" variant="ghost" onClick={onClose} className="[&_svg]:size-3">
            <IconXmarkOutline18 className="text-grayA-9 stroke-2" />
          </Button>
        </div>

        {SECTION_BODY_HEIGHTS.map((height, index) => (
          <div
            // biome-ignore lint/suspicious/noArrayIndexKey: fixed-length placeholder list
            key={index}
            className="flex flex-col gap-1 mt-[16px] px-4"
          >
            <div className="border bg-gray-2 border-gray-4 rounded-[10px]">
              <div className="px-[14px] py-1.5">
                <Skeleton className="h-3 w-28" />
              </div>
              <div className="border-gray-4 border-t rounded-[10px] bg-white dark:bg-black px-3.5 py-2">
                <Skeleton className={`w-full ${height}`} />
              </div>
            </div>
          </div>
        ))}

        <div className="mt-3" />

        <div className="px-4 font-sans">
          {FOOTER_LABEL_WIDTHS.map((width, index) => (
            <div
              // biome-ignore lint/suspicious/noArrayIndexKey: fixed-length placeholder list
              key={index}
              className="flex w-full justify-between border-grayA-3 border-solid border-b pr-3 py-3 items-center"
            >
              <Skeleton className={`h-3 ${width}`} />
              <Skeleton className="h-3 w-32" />
            </div>
          ))}
        </div>
      </div>
    </ResizablePanel>
  );
};

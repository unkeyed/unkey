"use client";
import { IconArrowsAllDirectionsOutline18, IconArrowsToCenterOutline18 } from "@unkey/icons";
import { useCallback, useState } from "react";
import { cn } from "../../../../lib/utils";
import { Button } from "../../../buttons/button";
import { FOOTER_PANEL } from "../../constants/constants";

export interface LoadMoreFooterComponentProps {
  onLoadMore?: () => void;
  isFetchingNextPage?: boolean;
  totalVisible: number;
  totalCount: number;
  className?: string;
  itemLabel?: string;
  buttonText?: string;
  hasMore: boolean;
  hide?: boolean;
  countInfoText?: React.ReactNode;
  headerContent?: React.ReactNode;
}

/**
 * Load more footer component with collapsible design
 * Preserves exact design from virtual-table
 */
export function LoadMoreFooter({
  onLoadMore,
  isFetchingNextPage = false,
  totalVisible,
  totalCount,
  itemLabel = "items",
  buttonText = "Load more",
  hasMore,
  countInfoText,
  hide,
  headerContent,
}: LoadMoreFooterComponentProps) {
  const [isOpen, setIsOpen] = useState(true);

  const shouldShow = !!onLoadMore;

  const handleClose = useCallback(() => {
    setIsOpen(false);
  }, []);

  const handleOpen = useCallback(() => {
    setIsOpen(true);
  }, []);

  if (hide) {
    return null;
  }

  // Minimized state - parked at right side
  if (!isOpen) {
    return (
      <div className="fixed bottom-6 right-6 z-10 transition-all duration-300 ease-out animate-slide-in-from-bottom">
        <button
          type="button"
          onClick={handleOpen}
          className="bg-raised rounded-lg shadow-floating p-3 transition-all duration-200 hover:scale-105 group"
          title={`${buttonText} • ${totalVisible} of ${totalCount} ${itemLabel}`}
        >
          <div className="flex items-center gap-2">
            <div className="flex items-center gap-2">
              <span className="text-2xs text-gray-9 font-medium">{countInfoText}</span>
            </div>
            <div className="w-px h-3 bg-gray-6" />
            <span className="text-xs font-medium text-gray-11 group-hover:text-gray-12 transition-colors">
              {buttonText}
            </span>
            <span
              aria-hidden="true"
              className="inline-flex items-center justify-center [&_svg]:size-[14px] transition-all duration-200 rounded transform hover:scale-110"
            >
              <IconArrowsAllDirectionsOutline18 className="size-3" />
            </span>
          </div>
        </button>
      </div>
    );
  }

  return (
    <div
      className={cn(
        "fixed bottom-0 left-0 right-0 w-full items-center justify-center flex z-10 transition-all duration-300 ease-out pointer-events-none",
        shouldShow ? "opacity-100" : "opacity-0",
        shouldShow && isOpen && "animate-fade-slide-in",
      )}
    >
      <div
        className={cn(
          FOOTER_PANEL,
          "transition-all duration-200",
          shouldShow ? "pointer-events-auto" : "pointer-events-none",
        )}
        aria-hidden={!shouldShow}
      >
        <div className="flex flex-col w-full">
          {/* Header content */}
          {headerContent && <div className="flex items-center w-full">{headerContent}</div>}

          <div className="flex w-full justify-between items-center text-sm text-gray-9 p-[18px] transition-all duration-200 animate-fade-slide-in [animation-delay:0.3s] [animation-fill-mode:backwards]">
            {countInfoText && <div className="transition-all duration-200">{countInfoText}</div>}
            {!countInfoText && (
              <div className="flex gap-2 transition-all duration-200">
                <span>Viewing</span>
                <span className="text-gray-12 transition-colors duration-200">{totalVisible}</span>
                <span>of</span>
                <span className="text-grayA-12 transition-colors duration-200">{totalCount}</span>
                <span>{itemLabel}</span>
              </div>
            )}

            <div className="items-center flex gap-2">
              <Button
                variant="outline"
                size="sm"
                onClick={onLoadMore}
                loading={isFetchingNextPage}
                disabled={isFetchingNextPage || !hasMore}
                className="transition-all"
              >
                {buttonText}
              </Button>
              <div
                className="flex justify-end transition-all duration-200 animate-fade-in-down"
                style={{ animationDelay: "0.1s" }}
              >
                <Button
                  size="icon"
                  variant="ghost"
                  className="[&_svg]:size-[14px] transition-all duration-200 rounded hover:bg-gray-3 transform hover:scale-110"
                  onClick={handleClose}
                  title="Minimize"
                >
                  <IconArrowsToCenterOutline18 className="size-4" />
                </Button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

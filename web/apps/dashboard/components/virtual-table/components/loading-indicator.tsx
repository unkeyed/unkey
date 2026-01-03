import { ArrowsToAllDirections, ArrowsToCenter } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { useCallback, useState } from "react";

type LoadMoreFooterProps = {
  onLoadMore?: () => void;
  isFetchingNextPage?: boolean;
  totalVisible: number;
  totalCount: number;
  className?: string;
  itemLabel?: string;
  buttonText?: string;
  hasMore?: boolean;
  hide?: boolean;
  countInfoText?: React.ReactNode;
  headerContent?: React.ReactNode;
};

export const LoadMoreFooter = ({
  onLoadMore,
  isFetchingNextPage = false,
  totalVisible,
  totalCount,
  itemLabel = "items",
  buttonText = "Load more",
  hasMore = true,
  countInfoText,
  hide,
  headerContent,
}: LoadMoreFooterProps) => {
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
      <div
        className="fixed bottom-6 right-6 z-10 transition-all duration-300 ease-out"
        style={{
          animation: "slideInFromBottom 0.3s ease-out",
        }}
      >
        <button
          type="button"
          onClick={handleOpen}
          className="bg-gray-1 dark:bg-black border border-gray-6 rounded-lg shadow-lg p-3 transition-all duration-200 hover:shadow-xl hover:scale-105 group"
          title={`${buttonText} • ${totalVisible} of ${totalCount} ${itemLabel}`}
        >
          <div className="flex items-center gap-2">
            <div className="flex items-center gap-2">
              <span className="text-[11px] text-gray-9 font-medium">{countInfoText}</span>
            </div>
            <div className="w-px h-3 bg-gray-6" />
            <span className="text-[12px] font-medium text-gray-11 group-hover:text-gray-12 transition-colors">
              {buttonText}
            </span>
            <Button
              size="icon"
              variant="ghost"
              className="[&_svg]:size-[14px] transition-all duration-200 rounded hover:bg-gray-3 transform hover:scale-110"
              title="Maximize"
            >
              <ArrowsToAllDirections iconSize="sm-regular" />
            </Button>
          </div>
        </button>
      </div>
    );
  }

  return (
    <div
      className="fixed bottom-0 left-0 right-0 w-full items-center justify-center flex z-10 transition-all duration-300 ease-out pointer-events-none"
      style={{
        opacity: shouldShow ? 1 : 0,
        animation: isOpen ? "slideUpFromBottom 0.3s ease-out" : undefined,
      }}
    >
      <div
        className={`w-[740px] border bg-gray-1 dark:bg-black border-gray-6 min-h-[60px] flex items-center justify-center rounded-[10px] drop-shadow-lg transform-gpu shadow-sm mb-5 transition-all duration-200 hover:shadow-lg ${
          shouldShow ? "pointer-events-auto" : "pointer-events-none"
        }`}
        aria-hidden={!shouldShow}
      >
        <div className="flex flex-col w-full">
          {/* Header content */}
          {headerContent && (
            <div
              className="transition-all duration-200"
              style={{
                animation: "fadeInUp 0.3s ease-out 0.2s both",
              }}
            >
              {headerContent}
            </div>
          )}

          <div
            className="flex w-full justify-between items-center text-[13px] text-accent-9 p-[18px] transition-all duration-200"
            style={{
              animation: "fadeInUp 0.3s ease-out 0.3s both",
            }}
          >
            {countInfoText && <div className="transition-all duration-200">{countInfoText}</div>}
            {!countInfoText && (
              <div className="flex gap-2 transition-all duration-200">
                <span>Viewing</span>
                <span className="text-accent-12 transition-colors duration-200">
                  {totalVisible}
                </span>
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
                className="flex justify-end transition-all duration-200"
                style={{
                  animation: "fadeInDown 0.3s ease-out 0.1s both",
                }}
              >
                <Button
                  size="icon"
                  variant="ghost"
                  className="[&_svg]:size-[14px] transition-all duration-200 rounded hover:bg-gray-3 transform hover:scale-110"
                  onClick={handleClose}
                  title="Minimize"
                >
                  <ArrowsToCenter iconSize="lg-regular" />
                </Button>
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* CSS Keyframes */}
      <style jsx>{`
        @keyframes slideUpFromBottom {
          from {
            opacity: 0;
            transform: translateY(100%);
          }
          to {
            opacity: 1;
            transform: translateY(0);
          }
        }

        @keyframes slideInFromBottom {
          from {
            opacity: 0;
            transform: translateY(20px) scale(0.95);
          }
          to {
            opacity: 1;
            transform: translateY(0) scale(1);
          }
        }

        @keyframes fadeInDown {
          from {
            opacity: 0;
            transform: translateY(-10px);
          }
          to {
            opacity: 1;
            transform: translateY(0);
          }
        }

        @keyframes fadeInUp {
          from {
            opacity: 0;
            transform: translateY(10px);
          }
          to {
            opacity: 1;
            transform: translateY(0);
          }
        }
      `}</style>
    </div>
  );
};

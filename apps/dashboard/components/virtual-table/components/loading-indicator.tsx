import { Button } from "@unkey/ui";

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
}: LoadMoreFooterProps) => {
  const shouldShow = !!onLoadMore;

  if (hide) {
    return;
  }

  return (
    <div
      className="fixed bottom-0 left-0 right-0 w-full items-center justify-center flex z-10 transition-opacity duration-200"
      style={{
        opacity: shouldShow ? 1 : 0,
        pointerEvents: shouldShow ? "auto" : "none",
      }}
    >
      <div className="w-[740px] border bg-gray-1 dark:bg-black border-gray-6 h-[60px] flex items-center justify-center p-[18px] rounded-[10px] drop-shadow-lg shadow-sm mb-5">
        <div className="flex w-full justify-between items-center text-[13px] text-accent-9">
          {countInfoText && <div>{countInfoText}</div>}
          {!countInfoText && (
            <div className="flex gap-2">
              <span>Viewing</span> <span className="text-accent-12">{totalVisible}</span>
              <span>of</span>
              <span className="text-grayA-12">{totalCount}</span>
              <span>{itemLabel}</span>
            </div>
          )}

          <Button
            variant="outline"
            size="sm"
            onClick={onLoadMore}
            loading={isFetchingNextPage}
            disabled={isFetchingNextPage || !hasMore}
          >
            {buttonText}
          </Button>
        </div>
      </div>
    </div>
  );
};

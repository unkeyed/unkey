import { CopyButton } from "@/components/dashboard/copy-button";
import type { Identity } from "@/lib/db";
import { User } from "@unkey/icons";
import { Button, Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@unkey/ui";

type IdentitySelectorProps = {
  identities: Omit<Identity, "deleted">[];
  hasNextPage?: boolean;
  isFetchingNextPage: boolean;
  loadMore: () => void;
};

const isMetaEmpty = (meta: unknown) => {
  if (!meta) {
    return true;
  }
  if (typeof meta !== "object") {
    return false;
  }
  return Object.keys(meta).length === 0;
};

export function createIdentityOptions({
  identities,
  hasNextPage,
  isFetchingNextPage,
  loadMore,
}: IdentitySelectorProps) {
  const options = identities.map((identity) => ({
    label: (
      <TooltipProvider>
        <Tooltip>
          <TooltipTrigger asChild>
            <div className="flex w-full text-accent-8 text-xs gap-1.5 py-0.5 items-center group">
              <div className="flex items-center justify-center gap-2">
                <div className="border rounded-full flex items-center justify-center border-grayA-6 size-5">
                  <User size="sm-regular" className="text-grayA-11" />
                </div>
                <span className="text-accent-12 font-medium text-xs w-[120px] truncate text-left">
                  {identity.id}
                </span>
              </div>
              <span className="w-[200px] truncate text-accent-8 group-hover:text-accent-9 text-left">
                {identity.externalId}
              </span>
            </div>
          </TooltipTrigger>
          <TooltipContent
            side="right"
            align="start"
            sideOffset={30}
            className="drop-shadow-2xl border border-gray-4 overflow-hidden rounded-[10px] p-0 bg-white dark:bg-black w-[320px] z-[100]"
          >
            <div className="flex flex-col h-full">
              {/* Header - Always shown */}
              <div className="px-4 py-2 border-b border-grayA-4 text-gray-10 text-xs font-medium bg-grayA-2">
                Metadata
              </div>
              {/* Content - Different based on metadata presence */}
              {!isMetaEmpty(identity.meta) ? (
                <div className="px-2 py-2 flex-1 overflow-y-auto h-[270px]">
                  <div className="w-full bg-grayA-1 dark:bg-grayA-2 border rounded-lg border-grayA-5 overflow-hidden h-full">
                    <div className="flex items-start justify-between w-full gap-2 h-full">
                      {/* JSON Content */}
                      <div className="overflow-x-auto w-full min-w-0 p-3 h-full">
                        <pre className="whitespace-pre-wrap break-all text-[11px] leading-5 text-gray-12 font-mono h-full overflow-y-auto">
                          {JSON.stringify(identity.meta, null, 4)}
                        </pre>
                      </div>
                      {/* Copy Button */}
                      <div className="p-2 flex-shrink-0">
                        <Button
                          variant="outline"
                          size="icon"
                          className="bg-white dark:bg-grayA-3 hover:bg-grayA-3 dark:hover:bg-grayA-4 shadow-sm"
                        >
                          <div className="flex items-center justify-center">
                            <CopyButton value={JSON.stringify(identity.meta, null, 4)} />
                          </div>
                        </Button>
                      </div>
                    </div>
                  </div>
                </div>
              ) : (
                <div className="px-2 py-2 flex-1">
                  <div className="w-full bg-grayA-1 dark:bg-grayA-2 border rounded-lg border-grayA-5 overflow-hidden">
                    <div className="flex items-start justify-between w-full gap-2">
                      <div className="overflow-x-auto w-full min-w-0 p-3">
                        <pre className="whitespace-pre-wrap break-all text-[11px] leading-5 text-gray-8 font-mono">
                          No metadata available
                        </pre>
                      </div>
                    </div>
                  </div>
                </div>
              )}
            </div>
          </TooltipContent>
        </Tooltip>
      </TooltipProvider>
    ),
    selectedLabel: (
      <div className="flex w-full text-accent-8 text-xs gap-1.5 py-0.5 items-center">
        <div className="flex items-center justify-center gap-2">
          <div className="border rounded-full flex items-center justify-center border-grayA-6 size-5">
            <User size="sm-regular" className="text-grayA-11" />
          </div>
          <span className="text-accent-12 font-medium text-xs w-[120px] truncate text-left">
            {identity.id}
          </span>
        </div>
        <span className="w-[200px] truncate text-accent-8 text-left">{identity.externalId}</span>
      </div>
    ),
    value: identity.id,
    searchValue: identity.externalId,
  }));

  if (hasNextPage) {
    options.push({
      label: (
        <Button
          type="button"
          variant="ghost"
          loading={isFetchingNextPage}
          onClick={(e) => {
            e.preventDefault();
            e.stopPropagation();
            loadMore();
          }}
          className="text-xs text-accent-12 px-2 py-0.5 hover:bg-grayA-3 rounded w-full bg-transparent hover:bg-transparent focus:ring-0 font-medium"
        >
          Load more...
        </Button>
      ),
      value: "__load_more__",
      selectedLabel: <></>,
      searchValue: "",
    });
  }

  return options;
}

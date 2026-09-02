import type { Identity } from "@unkey/api/models/components";
import { CopyButton, Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@unkey/ui";
import { IconUserOutline12 } from "nucleo-ui-outline-12";

type IdentitySelectorProps = {
  identities: Identity[];
  hasNextPage?: boolean;
  isFetchingNextPage: boolean;
  queryError?: string;
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
  queryError,
}: IdentitySelectorProps) {
  const options = identities.map((identity) => ({
    label: (
      <TooltipProvider>
        <Tooltip>
          <TooltipTrigger
            render={
              <div className="flex w-full text-accent-8 text-xs gap-1.5 py-0.5 items-center group">
                <div className="flex items-center justify-center gap-2">
                  <div className="border rounded-full flex items-center justify-center border-grayA-6 size-5">
                    <IconUserOutline12 className="text-grayA-11" />
                  </div>
                  <span className="max-w-[200px] truncate font-normal text-accent-12 text-left">
                    {identity.externalId.length > 15
                      ? `${identity.externalId.slice(0, 4)}...${identity.externalId.slice(-4)}`
                      : identity.externalId}
                  </span>
                </div>
              </div>
            }
          />
          <TooltipContent
            side="right"
            align="start"
            sideOffset={30}
            className="drop-shadow-2xl transform-gpu border border-grayA-4 overflow-hidden rounded-[10px] p-0 bg-white dark:bg-black w-[320px] z-100"
          >
            <div className="flex flex-col h-full">
              {/* Header - Always shown */}
              <div className="px-4 py-2 border-b border-grayA-4 text-gray-10 text-xs font-normal bg-grayA-2">
                Metadata
              </div>
              {/* Content - Different based on metadata presence */}
              {isMetaEmpty(identity.meta) ? (
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
              ) : (
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
                      <div className="p-2 shrink-0">
                        <CopyButton
                          value={JSON.stringify(identity.meta, null, 4)}
                          className="bg-white dark:bg-grayA-3 hover:bg-grayA-3 dark:hover:bg-grayA-4 shadow-xs"
                        />
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
            <IconUserOutline12 className="text-grayA-11" />
          </div>
          <span className="text-accent-12 font-normal text-xs max-w-[200px] truncate text-left">
            {identity.externalId}
          </span>
        </div>
      </div>
    ),
    value: identity.id,
    searchValue: identity.externalId,
  }));

  if (queryError) {
    options.push({
      label: (
        <div className="flex w-full flex-col gap-1 px-2 py-1 text-left">
          <span className="text-xs text-error-11">{queryError}</span>
          <span className="text-xs font-normal text-accent-11">Select to retry</span>
        </div>
      ),
      value: "__retry_identities__",
      selectedLabel: <></>,
      searchValue: "",
    });
  } else if (hasNextPage) {
    options.push({
      label: (
        <div className="w-full px-2 py-0.5 text-left text-xs font-normal text-accent-12">
          {isFetchingNextPage ? "Loading more..." : "Load more..."}
        </div>
      ),
      value: "__load_more__",
      selectedLabel: <></>,
      searchValue: "",
    });
  }

  return options;
}

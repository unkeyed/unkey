import { LoadMoreFooter } from "@/components/virtual-table/components/loading-indicator";
import { Bookmark } from "@unkey/icons";
import { Button, CopyButton, Empty } from "@unkey/ui";
import { useNamespaceListQuery } from "./hooks/use-namespace-list-query";
import { NamespaceCard } from "./namespace-card";
import { NamespaceCardSkeleton } from "./skeletons";

const MAX_SKELETON_COUNT = 10;
const MINIMUM_DISPLAY_LIMIT = 10;

const EXAMPLE_SNIPPET = `curl -XPOST 'https://api.unkey.dev/v2/ratelimits.limit' \\
  -H 'Content-Type: application/json' \\
  -H 'Authorization: Bearer <UNKEY_ROOT_KEY>' \\
  -d '{
      "namespace": "demo_namespace",
      "identifier": "user_123",
      "limit": 10,
      "duration": 10000
  }'`;

export const NamespaceList = () => {
  const {
    projects: namespaces,
    isLoading,
    totalCount,
    hasMore,
    loadMore,
    isLoadingMore,
  } = useNamespaceListQuery();

  if (isLoading) {
    return (
      <div className="p-4">
        <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-4 gap-3 md:gap-5">
          {Array.from({ length: MAX_SKELETON_COUNT }).map((_, i) => (
            // biome-ignore lint/suspicious/noArrayIndexKey: skeleton items don't need stable keys
            <NamespaceCardSkeleton key={i} />
          ))}
        </div>
      </div>
    );
  }

  if (namespaces.length === 0) {
    return (
      <div className="w-full flex justify-center items-center h-full p-4">
        <Empty className="w-[600px] flex items-start">
          <Empty.Icon />
          <Empty.Title>No Namespaces found</Empty.Title>
          <Empty.Description className="text-left">
            You haven't created any Namespaces yet. Create one by performing a limit request as
            shown below.
          </Empty.Description>
          <div className="w-full mt-8 mb-8">
            <div className="flex items-start gap-4 p-4 bg-gray-2 border border-gray-6 rounded-lg">
              <pre className="flex-1 text-xs text-left overflow-x-auto">
                <code>{EXAMPLE_SNIPPET}</code>
              </pre>
              <CopyButton value={EXAMPLE_SNIPPET} />
            </div>
          </div>
          <Empty.Actions className="mt-4 justify-start">
            <a href="/docs/ratelimiting/introduction" target="_blank" rel="noopener noreferrer">
              <Button className="flex items-center gap-2">
                <Bookmark className="w-4 h-4" />
                Read the docs
              </Button>
            </a>
          </Empty.Actions>
        </Empty>
      </div>
    );
  }

  return (
    <>
      <div className="p-4">
        <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-4 gap-3 md:gap-5">
          {namespaces.map((namespace) => (
            <NamespaceCard namespace={namespace} key={namespace.id} />
          ))}
        </div>
      </div>
      {totalCount > MINIMUM_DISPLAY_LIMIT ? (
        <LoadMoreFooter
          onLoadMore={loadMore}
          isFetchingNextPage={isLoadingMore}
          totalVisible={namespaces.length}
          totalCount={totalCount}
          itemLabel="namespaces"
          buttonText="Load more namespaces"
          hasMore={hasMore}
          hide={!hasMore && namespaces.length === totalCount}
          countInfoText={
            <div className="flex gap-2">
              <span>Viewing</span>
              <span className="text-accent-12">{namespaces.length}</span>
              <span>of</span>
              <span className="text-grayA-12">{totalCount}</span>
              <span>Namespaces</span>
            </div>
          }
        />
      ) : null}
    </>
  );
};

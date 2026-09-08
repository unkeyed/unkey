import { StatsListCardSkeleton } from "@/components/stats-list-card/skeleton";
import { collection } from "@/lib/collections";
import { ilike, useLiveQuery } from "@tanstack/react-db";
import { Bookmark } from "@unkey/icons";
import { Button, CopyButton, Empty, ResourceListContent } from "@unkey/ui";
import { useMemo } from "react";
import { useBatchRatelimitTimeseries } from "../hooks/use-batch-timeseries";
import { useNamespaceListFilters } from "../hooks/use-namespace-list-filters";
import { NamespaceCard } from "./namespace-card";

const SKELETON_COUNT = 8;

const EXAMPLE_SNIPPET = `curl -XPOST 'https://api.unkey.dev/v2/ratelimit.limit' \\
  -H 'Content-Type: application/json' \\
  -H 'Authorization: Bearer <UNKEY_ROOT_KEY>' \\
  -d '{
      "namespace": "demo_namespace",
      "identifier": "user_123",
      "limit": 10,
      "duration": 10000
  }'`;

export const NamespaceList = () => {
  const { filters } = useNamespaceListFilters();

  const nameFilter = filters.find((filter) => filter.field === "query")?.value ?? "";

  const { data: namespaces, isLoading: namespacesLoading } = useLiveQuery(
    (q) =>
      q
        .from({ namespace: collection.ratelimitNamespaces })
        .where(({ namespace }) => ilike(namespace.name, `%${nameFilter}%`))
        .orderBy(({ namespace }) => namespace.id, "desc"),
    [nameFilter],
  );

  const namespaceIds = useMemo(() => namespaces.map((ns) => ns.id), [namespaces]);
  const { timeseriesByNamespace, isLoading, isError } = useBatchRatelimitTimeseries(namespaceIds);

  if (namespacesLoading) {
    return (
      <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-5 w-full">
        {Array.from({ length: SKELETON_COUNT }).map((_, i) => (
          // biome-ignore lint/suspicious/noArrayIndexKey: skeleton items don't need stable keys
          <StatsListCardSkeleton key={i} />
        ))}
      </div>
    );
  }

  if (namespaces.length === 0) {
    return (
      <ResourceListContent>
        <div className="flex w-full items-center justify-center px-4 py-16">
          <Empty className="w-[600px] items-start p-0">
            <Empty.Icon className="w-auto" />
            <Empty.Title>No Namespaces found</Empty.Title>
            <Empty.Description className="text-left">
              You haven't created any Namespaces yet. Create one by performing a limit request as
              shown below.
            </Empty.Description>
            <div className="w-full mt-6">
              <div className="flex items-start gap-4 p-4 bg-gray-2 border border-gray-6 rounded-lg">
                <pre className="flex-1 text-xs text-left overflow-x-auto">
                  <code>{EXAMPLE_SNIPPET}</code>
                </pre>
                <CopyButton value={EXAMPLE_SNIPPET} />
              </div>
            </div>
            <Empty.Actions className="mt-4 justify-start">
              <a
                href="https://www.unkey.com/docs/platform/ratelimiting/introduction"
                target="_blank"
                rel="noopener noreferrer"
              >
                <Button className="flex items-center gap-2">
                  <Bookmark className="w-4 h-4" />
                  Read the docs
                </Button>
              </a>
            </Empty.Actions>
          </Empty>
        </div>
      </ResourceListContent>
    );
  }

  return (
    <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-5 w-full">
      {namespaces.map((namespace) => (
        <NamespaceCard
          namespace={namespace}
          key={namespace.id}
          timeseries={timeseriesByNamespace[namespace.id]}
          isLoading={isLoading}
          isError={isError}
        />
      ))}
    </div>
  );
};

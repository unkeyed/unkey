import { ListPager } from "@/components/list-pager";
import { LoadError } from "@/components/load-error";
import { StatsListCardSkeleton } from "@/components/stats-list-card/skeleton";
import { useInfiniteList } from "@/lib/loadable/list";
import { useNamespaces } from "@/lib/queries/ratelimit-namespaces";
import { IconBook2Outline18, IconGaugeOutline18 } from "@unkey/icons";
import { match } from "@unkey/match";
import {
  Button,
  CopyButton,
  EmptyState,
  EmptyStateActions,
  EmptyStateDescription,
  EmptyStateHeader,
  EmptyStateIcon,
  EmptyStateTitle,
} from "@unkey/ui";
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

  const view = useInfiniteList(
    useNamespaces({ search: String(nameFilter).trim() }),
    (page) => page.namespaces,
  );

  const namespaceIds = view.status === "ready" ? view.rows.map((ns) => ns.id) : [];
  const { timeseriesByNamespace, isLoading, isError } = useBatchRatelimitTimeseries(namespaceIds);

  return match(view)
    .with({ status: "loading" }, () => (
      <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-5 w-full">
        {Array.from({ length: SKELETON_COUNT }).map((_, i) => (
          // biome-ignore lint/suspicious/noArrayIndexKey: skeleton items don't need stable keys
          <StatsListCardSkeleton key={i} />
        ))}
      </div>
    ))
    .with({ status: "failed" }, ({ error, retry }) => (
      <LoadError title="Could not load namespaces" error={error} onRetry={retry} />
    ))
    .with({ status: "empty" }, () => <NamespacesEmpty />)
    .with({ status: "ready" }, ({ rows, footer }) => (
      <div className="flex flex-col items-center gap-6 w-full">
        <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-5 w-full">
          {rows.map((namespace) => (
            <NamespaceCard
              namespace={namespace}
              key={namespace.id}
              timeseries={timeseriesByNamespace[namespace.id]}
              isLoading={isLoading}
              isError={isError}
            />
          ))}
        </div>
        <ListPager state={footer} noun="namespaces" className="justify-center border-t-0 p-0" />
      </div>
    ))
    .exhaustive();
};

function NamespacesEmpty() {
  return (
    <EmptyState>
      <EmptyStateIcon>
        <IconGaugeOutline18 />
      </EmptyStateIcon>
      <EmptyStateHeader>
        <EmptyStateTitle>No Namespaces found</EmptyStateTitle>
        <EmptyStateDescription>
          You haven't created any Namespaces yet. Create one by performing a limit request as shown
          below.
        </EmptyStateDescription>
      </EmptyStateHeader>
      <div className="mt-4 w-full max-w-lg">
        <div className="flex items-start gap-4 rounded-lg border bg-background p-4">
          <pre className="flex-1 text-xs text-left overflow-x-auto">
            <code>{EXAMPLE_SNIPPET}</code>
          </pre>
          <CopyButton value={EXAMPLE_SNIPPET} />
        </div>
      </div>
      <EmptyStateActions>
        <a
          href="https://www.unkey.com/docs/platform/ratelimiting/introduction"
          target="_blank"
          rel="noopener noreferrer"
        >
          <Button variant="outline" className="flex items-center gap-2">
            <IconBook2Outline18 className="w-4 h-4" />
            Read the docs
          </Button>
        </a>
      </EmptyStateActions>
    </EmptyState>
  );
}

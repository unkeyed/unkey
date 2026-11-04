import { PAGINATED_LIST_QUERY_OPTIONS } from "@/hooks/use-paginated-list-query";
import { type ShownRows, shownTotalOf, tableStateOf, useInfiniteList } from "@/lib/loadable/list";
import { type QueryKey, useInfiniteQuery } from "@tanstack/react-query";

type LoadMoreListConfig<TPage, TRow> = {
  queryKey: QueryKey;
  fetchPage: (page: number, signal: AbortSignal | undefined) => Promise<TPage>;
  rowsOf: (page: TPage) => TRow[];
  pageSize: number;
  paused?: boolean;
} & ShownRows<TRow>;

export function useLoadMoreList<TPage, TRow>(config: LoadMoreListConfig<TPage, TRow>) {
  const { queryKey, fetchPage, rowsOf, idOf, mapRows, pageSize, paused = false } = config;

  const query = useInfiniteQuery({
    queryKey,
    queryFn: ({ pageParam = 1, signal }) => fetchPage(pageParam, signal),
    getNextPageParam: (lastPage, allPages) => {
      const hasMore = !paused && rowsOf(lastPage).length >= pageSize;
      return hasMore ? allPages.length + 1 : undefined;
    },
    ...PAGINATED_LIST_QUERY_OPTIONS,
    keepPreviousData: false,
  });

  const table = tableStateOf(useInfiniteList(query, rowsOf, { idOf, mapRows }));

  return {
    table,
    totalCount: shownTotalOf({
      loaded: table.rows.length,
      total: null,
      hasMore: paused || query.hasNextPage === true,
    }),
  };
}

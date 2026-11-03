import { PAGINATED_LIST_QUERY_OPTIONS } from "@/hooks/use-paginated-list-query";
import {
  type ShownRows,
  shownTotalOf,
  tableStateOf,
  uniqueRowsOf,
  useInfiniteList,
} from "@/lib/loadable/list";
import { type QueryKey, useInfiniteQuery } from "@tanstack/react-query";

type TotalSource<TPage> =
  | { totalOf: (firstPage: TPage) => number | null | undefined }
  | { total: number | null };

type LoadMoreListConfig<TPage, TRow> = {
  queryKey: QueryKey;
  fetchPage: (page: number, signal: AbortSignal | undefined) => Promise<TPage>;
  rowsOf: (page: TPage) => TRow[];
  pageSize: number;
  paused?: boolean;
} & ShownRows<TRow> &
  TotalSource<TPage>;

function knownTotalOf<TPage>(pages: readonly TPage[], source: TotalSource<TPage>) {
  if ("total" in source) {
    return source.total;
  }
  return pages[0] === undefined ? null : (source.totalOf(pages[0]) ?? null);
}

export function useLoadMoreList<TPage, TRow>(config: LoadMoreListConfig<TPage, TRow>) {
  const { queryKey, fetchPage, rowsOf, idOf, mapRows, pageSize, paused = false } = config;

  const query = useInfiniteQuery({
    queryKey,
    queryFn: ({ pageParam = 1, signal }) => fetchPage(pageParam, signal),
    getNextPageParam: (lastPage, allPages) => {
      const lastRows = rowsOf(lastPage).length;
      if (paused || lastRows === 0) {
        return undefined;
      }
      const total = knownTotalOf(allPages, config);
      const all = allPages.flatMap(rowsOf);
      const loaded = idOf ? uniqueRowsOf(all, idOf).length : all.length;
      const hasMore = total === null ? lastRows >= pageSize : loaded < total;
      return hasMore ? allPages.length + 1 : undefined;
    },
    ...PAGINATED_LIST_QUERY_OPTIONS,
    keepPreviousData: false,
  });

  const table = tableStateOf(useInfiniteList(query, rowsOf, { idOf, mapRows }));
  const totalStaysOpen = paused || query.hasNextPage === true;

  return {
    table,
    totalCount: shownTotalOf({
      loaded: table.rows.length,
      total: knownTotalOf(query.data?.pages ?? [], config),
      hasMore: totalStaysOpen,
    }),
  };
}

import {
  rootKeysFilterFieldConfig,
  rootKeysListFilterFieldNames,
} from "@/app/(app)/[workspaceSlug]/settings/root-keys/filters.schema";
import type { RootKeysFilterValue } from "@/app/(app)/[workspaceSlug]/settings/root-keys/filters.schema";
import { useFilters } from "@/app/(app)/[workspaceSlug]/settings/root-keys/hooks/use-filters";
import { trpc } from "@/lib/trpc/client";

// Mirrors LIMIT in query.ts — kept here to avoid importing the server-side router into the client bundle
const DEFAULT_PAGE_SIZE = 50;
import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
  useTransition,
} from "react";
import type { RootKeysQueryPayload } from "../../schema/query-logs.schema";

function buildQueryParams(filters: RootKeysFilterValue[]): RootKeysQueryPayload {
  const params: RootKeysQueryPayload = {
    ...Object.fromEntries(rootKeysListFilterFieldNames.map((field) => [field, []])),
  };

  for (const filter of filters) {
    if (!rootKeysListFilterFieldNames.includes(filter.field) || !params[filter.field]) {
      continue;
    }

    const fieldConfig = rootKeysFilterFieldConfig[filter.field];
    if (!fieldConfig.operators.includes(filter.operator)) {
      throw new Error("Invalid operator");
    }

    if (typeof filter.value === "string") {
      params[filter.field]?.push({
        operator: filter.operator,
        value: filter.value,
      });
    }
  }

  return params;
}

export function useRootKeysListPaginated(pageSize = DEFAULT_PAGE_SIZE) {
  const { filters } = useFilters();
  const [page, setPage] = useState(1);
  const [isPending, startTransition] = useTransition();

  // Maps page number → cursor (createdAtM) needed to fetch that page.
  // Page 1 always starts from the beginning (undefined cursor).
  const pageCursors = useRef<Map<number, number | undefined>>(new Map([[1, undefined]]));

  // Reset pagination whenever filters change.
  // biome-ignore lint/correctness/useExhaustiveDependencies: filters is an intentional trigger — not consumed in the body but must cause this effect to re-run on change
  useEffect(() => {
    pageCursors.current = new Map([[1, undefined]]);
    setPage(1);
  }, [filters]);

  const baseParams = useMemo(() => buildQueryParams(filters), [filters]);

  const queryParams = useMemo(
    () => ({
      ...baseParams,
      cursor: pageCursors.current.get(page),
      limit: pageSize,
    }),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [baseParams, page, pageSize],
  );

  const { data, isLoading, isFetching } = trpc.settings.rootKeys.query.useQuery(queryParams, {
    staleTime: Number.POSITIVE_INFINITY,
    refetchOnMount: false,
    refetchOnWindowFocus: false,
  });

  // Cache the next page's cursor as soon as the current page resolves.
  if (data?.nextCursor !== undefined && !pageCursors.current.has(page + 1)) {
    pageCursors.current.set(page + 1, data.nextCursor);
  }

  const totalCount = data?.total ?? 0;
  const totalPages = Math.max(1, Math.ceil(totalCount / pageSize));

  const onPageChange = useCallback(
    (newPage: number) => {
      if (newPage < 1 || newPage > totalPages || !pageCursors.current.has(newPage)) {
        return;
      }

      startTransition(() => {
        setPage(newPage);
      });
    },
    [totalPages],
  );

  return {
    rootKeys: data?.keys ?? [],
    isLoading,
    isPending,
    isFetching,
    page,
    pageSize,
    totalPages,
    totalCount,
    onPageChange,
  };
}

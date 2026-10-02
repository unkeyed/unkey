import { useState } from "react";
import { type KeyRow, type Sort, type SortKey, sortRows } from "~/components/keys-table/key-row";

const DEFAULT_SORT: Sort = { key: "valid", desc: true };

export const PAGE_SIZES = [10, 25, 50];

export function useTablePaging(rows: KeyRow[], sortable: boolean) {
  const [sort, setSort] = useState<Sort>(DEFAULT_SORT);
  const [page, setPage] = useState(0);
  const [pageSize, setSize] = useState(PAGE_SIZES[0]);

  const toggleSort = (key: SortKey) => {
    setPage(0);
    setSort((prev) => (prev.key === key ? { key, desc: !prev.desc } : { key, desc: true }));
  };

  const setPageSize = (size: number) => {
    setSize(size);
    setPage(0);
  };

  const sorted = sortRows(rows, sortable ? sort : null);
  const pageCount = Math.max(1, Math.ceil(sorted.length / pageSize));
  const currentPage = Math.min(page, pageCount - 1);
  const pageStart = currentPage * pageSize;

  return {
    sort,
    toggleSort,
    pageRows: sorted.slice(pageStart, pageStart + pageSize),
    pageCount,
    currentPage,
    pageStart,
    pageSize,
    setPageSize,
    setPage,
    total: sorted.length,
  };
}

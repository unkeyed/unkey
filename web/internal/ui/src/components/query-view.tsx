import type { ReactNode } from "react";

type QueryViewProps<T> = {
  isLoading: boolean;
  isError?: boolean;
  data: T | null | undefined;
  loading: ReactNode;
  error?: ReactNode;
  empty?: ReactNode;
  isEmpty?: (data: T) => boolean;
  children: (data: T) => ReactNode;
};

/**
 * Renders one of four states for fetched data, checked in this order:
 * loading, error, empty, then `children(data)`. Nullish data counts as empty,
 * as does an empty array unless `isEmpty` says otherwise.
 */
function QueryView<T>({
  isLoading,
  isError = false,
  data,
  loading,
  error = null,
  empty = null,
  isEmpty = isEmptyArray,
  children,
}: QueryViewProps<T>) {
  if (isLoading) {
    return loading;
  }
  if (isError) {
    return error;
  }
  if (data === null || data === undefined || isEmpty(data)) {
    return empty;
  }
  return children(data);
}

function isEmptyArray(data: unknown): boolean {
  return Array.isArray(data) && data.length === 0;
}

export { QueryView };

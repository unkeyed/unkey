import type { ReactNode } from "react";

/**
 * The three states a fetch can be in, one at a time. Uses React Query's status
 * names (v4 `loading`, v5 `pending`) so a query result is a valid state as is.
 */
type QueryState<T> =
  | { status: "loading" | "pending" }
  | { status: "error" }
  | { status: "success"; data: T | null | undefined };

type QueryViewProps<T> = {
  state: QueryState<T>;
  loading: ReactNode;
  error?: ReactNode;
  empty?: ReactNode;
  isEmpty?: (data: T) => boolean;
  children: (data: T) => ReactNode;
};

/**
 * Renders the slot for the current state. Nullish data counts as empty, as does
 * an empty array unless `isEmpty` says otherwise.
 */
function QueryView<T>({
  state,
  loading,
  error = null,
  empty = null,
  isEmpty = isEmptyArray,
  children,
}: QueryViewProps<T>) {
  switch (state.status) {
    case "loading":
    case "pending":
      return loading;
    case "error":
      return error;
    case "success":
      return state.data === null || state.data === undefined || isEmpty(state.data)
        ? empty
        : children(state.data);
    default:
      return state satisfies never;
  }
}

/** Folds loading and error flags into a `QueryState`. Loading wins, then error. */
function queryState<T>(input: {
  isPending?: boolean;
  isLoading: boolean;
  isError?: boolean;
  data: T | null | undefined;
}): QueryState<T> {
  if (input.isPending ?? input.isLoading) {
    return { status: "loading" };
  }
  if (input.isError) {
    return { status: "error" };
  }
  return { status: "success", data: input.data };
}

function isEmptyArray(data: unknown): boolean {
  return Array.isArray(data) && data.length === 0;
}

export { QueryView, queryState, type QueryState };

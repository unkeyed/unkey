export type Failure = { error: unknown; retry: () => void };

export type Stale = Failure & { retrying: boolean };

export type Loadable<T> =
  | { status: "loading" }
  | ({ status: "failed" } & Failure)
  | { status: "ready"; data: T; stale: Stale | null };

export type QueryResult<T> = {
  data: T | undefined;
  error: unknown;
  isError: boolean;
  isFetching: boolean;
  refetch: () => Promise<unknown>;
};

export function fromQuery<Q extends QueryResult<T>, T = Exclude<Q["data"], undefined>>(
  query: Q,
): Loadable<T> {
  const retry = () => {
    query.refetch().catch(() => {});
  };
  if (query.data !== undefined) {
    return {
      status: "ready",
      data: query.data,
      stale: query.isError ? { error: query.error, retry, retrying: query.isFetching } : null,
    };
  }
  if (query.isError) {
    return { status: "failed", error: query.error, retry };
  }
  return { status: "loading" };
}

export function readyOr<T>(view: Loadable<T>, fallback: T): T {
  return view.status === "ready" ? view.data : fallback;
}

export type Blocker = { status: "loading" } | ({ status: "failed" } & Failure);

export function blockerOf(views: ReadonlyArray<Loadable<unknown>>): Blocker | null {
  const failed = views.flatMap((view) => (view.status === "failed" ? [view] : []));
  const first = failed.at(0);
  if (first) {
    return {
      status: "failed",
      error: first.error,
      retry: () => {
        for (const view of failed) {
          view.retry();
        }
      },
    };
  }
  return views.some((view) => view.status === "loading") ? { status: "loading" } : null;
}

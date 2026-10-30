export type Failure = { error: unknown; retry: () => void; retrying: boolean };

export type Loadable<T> =
  | { status: "loading" }
  | ({ status: "failed" } & Failure)
  | { status: "ready"; data: T; stale: Failure | null };

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
    void query.refetch();
  };
  if (query.data !== undefined) {
    return {
      status: "ready",
      data: query.data,
      stale: query.isError ? { error: query.error, retry, retrying: query.isFetching } : null,
    };
  }
  if (query.isError) {
    return { status: "failed", error: query.error, retry, retrying: query.isFetching };
  }
  return { status: "loading" };
}

export function mapLoadable<T, U>(loadable: Loadable<T>, map: (data: T) => U): Loadable<U> {
  return loadable.status === "ready"
    ? { status: "ready", data: map(loadable.data), stale: loadable.stale }
    : loadable;
}

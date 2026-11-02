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

export function readyLoadable<T>(data: T): Loadable<T> {
  return { status: "ready", data, stale: null };
}

export function mapLoadable<T, U>(loadable: Loadable<T>, map: (data: T) => U): Loadable<U> {
  return loadable.status === "ready"
    ? { status: "ready", data: map(loadable.data), stale: loadable.stale }
    : loadable;
}

function failureOf<T>(loadable: Loadable<T>): Failure | null {
  if (loadable.status === "failed") {
    return loadable;
  }
  return loadable.status === "ready" ? loadable.stale : null;
}

function alsoRetry<F extends Failure>(failure: F, other: Failure | null): F {
  if (!other) {
    return failure;
  }
  return {
    ...failure,
    retry: () => {
      failure.retry();
      other.retry();
    },
    retrying: failure.retrying || other.retrying,
  };
}

export function zipLoadable<A, B, U>(
  a: Loadable<A>,
  b: Loadable<B>,
  combine: (a: A, b: B) => U,
): Loadable<U> {
  if (a.status === "loading") {
    return a;
  }
  if (a.status === "failed") {
    return alsoRetry(a, failureOf(b));
  }
  if (b.status === "loading") {
    return b;
  }
  if (b.status === "failed") {
    return alsoRetry(b, a.stale);
  }
  return {
    status: "ready",
    data: combine(a.data, b.data),
    stale: a.stale ? alsoRetry(a.stale, b.stale) : b.stale,
  };
}

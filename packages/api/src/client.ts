import type { UnkeyError } from "./errors";

type UnkeyRootKeyOptions = {
  /**
   * The root key from unkey.dev.
   *
   * You can create/manage your root keys here:
   * https://unkey.dev/app/settings/root-keys
   */
  rootKey: string;
};
type UnkeyTokenOptions = {
  /**
   * The workspace key from unkey.dev
   *
   * @deprecated Use `rootKey`
   */
  token: string;
};
type UnkeyBaseOptions = {
  /**
   * @default https://api.unkey.dev
   */
  baseUrl?: string;
  /**
   * Retry on network errors
   */
  retry?: Partial<Retry>;
  /**
   * Customize the `fetch` cache behaviour
   */
  cache?: RequestCache;
};

export type UnkeyOptions = (UnkeyRootKeyOptions | UnkeyTokenOptions) & UnkeyBaseOptions;

type Backoff = (retryCount: number) => number;
type Retry = {
  /**
   * How many attempts should be made
   * The maximum number of requests will be `attempts + 1`
   * `0` means no retries
   *
   * @default 5
   */
  attempts: number;
  /**
   * Return how many milliseconds to wait until the next attempt is made
   *
   * @default `(retryCount) => Math.round(Math.exp(retryCount) * 10)),`
   */
  backoff: Backoff;
};
type RateLimit = {
  type: "fast" | "consistent";
  /**
   * The total amount of burstable requests.
   */
  limit: number;

  /**
   * How many tokens to refill during each refillInterval
   */
  refillRate: number;

  /**
   * Determines the speed at which tokens are refilled.
   * In milliseconds.
   */
  refillInterval: number;
};
type RateLimitResult = {
  /**
   * The maximum number of requests for bursting.
   */
  limit: number;

  /**
   * How many requests are remaining until `reset`
   */
  remaining: number;

  /**
   * Unix timestamp in millisecond when the ratelimit is refilled.
   */
  reset: number;
};
type MetaRecord<T = unknown> = {
  /**
   * This is a place for dynamic meta data, anything that feels useful for you should go here
   *
   * Example:
   *
   * ```json
   * {
   *   "billingTier":"PRO",
   *   "trialEnds": "2023-06-16T17:16:37.161Z"
   * }
   * ```
   */
  meta?: T;
};

type CreateKeyPayload = {
  /**
   * Provide a name to this key if you want for later reference
   */
  name?: string;
  /**
   * Choose an API where this key should be created.
   */
  apiId: string;
  /**
   * To make it easier for your users to understand which product an api key belongs to, you can add prefix them.
   *
   * For example Stripe famously prefixes their customer ids with cus_ or their api keys with sk_live_.
   *
   * The underscore is automtically added if you are defining a prefix, for example: "prefix": "abc" will result in a key like abc_xxxxxxxxx
   */
  prefix?: string;

  /**
   * The bytelength used to generate your key determines its entropy as well as its length. Higher is better, but keys become longer and more annoying to handle.
   *
   * The default is 16 bytes, or 2128 possible combinations
   */
  byteLength?: number;
  /**
   * Your user’s Id. This will provide a link between Unkey and your customer record.
   *
   * When validating a key, we will return this back to you, so you can clearly identify your user from their api key.
   */
  ownerId?: string;
  /**
   * You can auto expire keys by providing a unix timestamp in milliseconds.
   *
   * Once keys expire they will automatically be deleted and are no longer valid.
   */
  expires?: number;

  /**
   * Unkey comes with per-key ratelimiting out of the box.
   *
   * @see https://unkey.dev/docs/features/ratelimiting
   */
  ratelimit?: RateLimit;

  /**
   * Unkey allows you to set/update usage limits on individual keys
   *
   * @see https://unkey.dev/docs/features/remaining
   */
  remaining?: number;
} & MetaRecord;
type UpdateKeyPayload = {
  /**
   * The id of the key to update.
   */
  keyId: string;
  /**
   * Update the name
   */
  name?: string | null;

  /**
   * Update the owner id
   */
  ownerId?: string | null;
  /**
   * Update the expiration time, Unix timstamp in milliseconds
   *
   *
   */
  expires?: number | null;

  /**
   * Update the ratelimit
   *
   * @see https://unkey.dev/docs/features/ratelimiting
   */
  ratelimit?: RateLimit | null;

  /**
   * Update the remaining verifications.
   *
   * @see https://unkey.dev/docs/features/remaining
   */
  remaining?: number | null;
} & MetaRecord;
export type VerifyKeyPayload = {
  /**
   * The key to verify
   */
  key: string;

  /**
   * The api id to verify against
   *
   * This will be required soon.
   */
  apiId?: string;
};
export type VerifyKeyResult = {
  /**
   * Whether or not this key is valid and has passed the ratelimit. If false you should not grant access to whatever the user is requesting
   */
  valid: boolean;

  /**
   * If you have set an ownerId on this key it is returned here. You can use this to clearly authenticate a user in your system.
   */
  ownerId?: string;

  /**
   *  Unix timestamp in milliseconds when this key expires
   * Only available when the key automatically expires
   */
  expires?: number;

  /**
   * How many verifications are remaining after the current request.
   */
  remaining?: number;

  /**
   * Ratelimit data if the key is ratelimited.
   */
  ratelimit?: RateLimitResult;

  /**
   * Machine readable code that explains why a key is invalid or could not be verified
   */
  code?: "NOT_FOUND" | "FORBIDDEN" | "RATELIMITED" | "KEY_USAGE_EXCEEDED";
} & MetaRecord;
type RevokeKeyPayload = { keyId: string };
type KeysResult = { key: string; keyId: string };

type ApiIdRecord = {
  /**
   * The global unique identifier of your api.
   */
  apiId: string;
};
type Key = {
  id: string;
  apiId: string;
  ownerId?: string;
  workspaceId: string;
  start: string;
  createdAt: number;
  name?: string;
  expires?: number;
  remaining?: number;
  ratelimit?: RateLimit;
} & MetaRecord;
type CreateApiPayload = {
  /**
   * A name for you to identify your API.
   */
  name: string;
};
type GetApiResult = { id: string; name: string; workspaceId: string };
type ListKeysPayload = ApiIdRecord & {
  /**
   * Limit the number of returned keys, the maximum is 100.
   *
   * @default 100
   */
  limit?: number;

  /**
   * Specify an offset for pagination.
   * @example:
   * An offset of 4 will skip the first 4 keys and return keys starting at the 5th position.
   *
   * @default 0
   */
  offset?: number;

  /**
   * If provided, this will only return keys where the ownerId matches.
   */
  ownerId?: string;
};
type ListKeysResult = {
  keys: Key[];
  total: number;
};

type CreateRootKeyPayload = {
  /**
   * Provide a name to this key if you want for later reference
   */
  name?: string;

  /**
   * You can auto expire keys by providing a unix timestamp in milliseconds.
   *
   * Once keys expire they will automatically be deleted and are no longer valid.
   */
  expires?: number;

  // Used to create root keys from the frontend, please ignore
  forWorkspaceId: string;
};
type DeleteRootKeyPayload = {
  /**
   *  Used to create root keys from the frontend, please ignore
   */
  keyId: string;
};

type SuccessResult<R> = {
  result: R;
  error?: never;
};
type ErrorResult = {
  result?: never;
  error: UnkeyError;
};
export type Result<R> = SuccessResult<R> | ErrorResult;
type ApiCall<P = unknown, R = unknown> = (payload: P) => Promise<Result<R>>;
type ApiRequest = {
  path: string[];
  query?: Record<string, string>;
} & (
  | { method: "GET" } // GET requests do not have a body
  | { method: "POST" | "PUT" | "DELETE"; body?: any }
);

export class Unkey {
  public readonly baseUrl: string;
  private readonly rootKey: string;
  private readonly cache?: RequestCache;

  public readonly retry: Retry;

  public constructor(opts: UnkeyOptions) {
    this.baseUrl = opts.baseUrl ?? "https://api.unkey.dev";
    this.rootKey = "rootKey" in opts ? opts.rootKey : opts.token;

    this.cache = opts.cache;
    /**
     * Even though typescript should prevent this, some people still pass undefined or empty strings
     */
    if (!this.rootKey) {
      throw new Error(
        "Unkey root key must be set, maybe you passed in `undefined` or an empty string?",
      );
    }

    this.retry = {
      attempts: opts.retry?.attempts ?? 5,
      backoff: opts.retry?.backoff ?? ((n) => Math.round(Math.exp(n) * 10)),
    } satisfies Retry;
  }

  private async fetch<TResult = unknown>(req: ApiRequest): Promise<Result<TResult>> {
    let res: Response | null = null;
    let err: Error | null = null;

    const url = new URL(req.path.join("/"), this.baseUrl);

    if (req.query) {
      for (const [k, v] of Object.entries(req.query)) {
        url.searchParams.set(k, v);
      }
    }

    for (let i = 0; i <= this.retry.attempts; i++) {
      try {
        const body = req.method === "GET" || !req.body ? undefined : JSON.stringify(req.body);

        res = await fetch(url, {
          method: req.method,
          headers: {
            "Content-Type": "application/json",
            Authorization: `Bearer ${this.rootKey}`,
          },
          cache: this.cache,
          body,
        });

        if (res.ok) {
          return { result: await res.json() } as SuccessResult<TResult>;
        }
      } catch (e) {
        err = e as Error;
        res = null; // set `res` to `null`
      }

      // no need to wait after last attempt failed
      if (i < this.retry.attempts) {
        const backoff = this.retry.backoff(i);
        console.debug(
          "attempt %d of %d to reach %s failed, retrying in %d ms: %s",
          i + 1,
          this.retry.attempts + 1,
          url,
          backoff,
          err?.message,
        );
        await new Promise<void>((r) => setTimeout(r, backoff));
      } else {
        console.debug(
          "attempt %d of %d to reach %s failed, giving up: %s",
          i + 1,
          this.retry.attempts + 1,
          url,
          err?.message,
        );
      }
    }

    if (res) {
      return { error: await res.json() } as ErrorResult;
    }

    return {
      error: {
        code: "FETCH_ERROR",
        message: err?.message ?? "No response",
        docs: "https://developer.mozilla.org/en-US/docs/Web/API/fetch",
        requestId: "N/A",
      },
    } satisfies ErrorResult;
  }

  public get keys(): {
    create: ApiCall<CreateKeyPayload, KeysResult>;
    update: ApiCall<UpdateKeyPayload, KeysResult>;
    verify: ApiCall<VerifyKeyPayload, VerifyKeyResult>;
    revoke: ApiCall<RevokeKeyPayload, void>;
  } {
    return {
      create: async (payload) =>
        this.fetch({
          path: ["v1", "keys"],
          method: "POST",
          body: payload,
        }),
      update: async (payload) =>
        this.fetch({
          path: ["v1", "keys", payload.keyId],
          method: "PUT",
          body: payload,
        }),
      verify: async (payload) =>
        this.fetch({
          path: ["v1", "keys", "verify"],
          method: "POST",
          body: payload,
        }),
      revoke: async (payload) =>
        this.fetch({
          path: ["v1", "keys", payload.keyId],
          method: "DELETE",
        }),
    };
  }

  public get apis(): {
    create: ApiCall<CreateApiPayload, ApiIdRecord>;
    remove: ApiCall<ApiIdRecord, ApiIdRecord>;
    get: ApiCall<ApiIdRecord, GetApiResult>;
    listKeys: ApiCall<ListKeysPayload, ListKeysResult>;
  } {
    return {
      create: async (payload) =>
        this.fetch({
          path: ["v1", "apis.createApi"],
          method: "POST",
          body: payload,
        }),
      remove: async (payload) =>
        this.fetch({
          path: ["v1", "apis.removeApi"],
          method: "POST",
          body: payload,
        }),
      get: async (payload) =>
        this.fetch({
          path: ["v1", "apis", payload.apiId],
          method: "GET",
        }),
      listKeys: async (payload) => {
        const query: Record<string, string> = {};
        if (typeof payload.limit !== "undefined") {
          query.limit = payload.limit.toString();
        }
        if (typeof payload.offset !== "undefined") {
          query.offset = payload.offset.toString();
        }
        if (typeof payload.ownerId !== "undefined") {
          query.ownerId = payload.ownerId;
        }

        return this.fetch({
          path: ["v1", "apis", payload.apiId, "keys"],
          method: "GET",
          query,
        });
      },
    };
  }

  /**
   * Must be authenticated via app token
   */
  public get _internal(): {
    createRootKey: ApiCall<CreateRootKeyPayload, KeysResult>;
    deleteRootKey: ApiCall<DeleteRootKeyPayload, void>;
  } {
    return {
      createRootKey: async (payload) =>
        this.fetch({
          path: ["v1", "internal", "rootkeys"],
          method: "POST",
          body: payload,
        }),
      deleteRootKey: async (payload) =>
        this.fetch({
          path: ["v1", "internal.removeRootKey"],
          method: "POST",
          body: payload,
        }),
    };
  }
}

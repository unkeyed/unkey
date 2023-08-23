import { UnkeyError } from "./errors";

export type UnkeyOptions = {
  /**
   * @default https://api.unkey.dev
   */
  baseUrl?: string;

  /**
   * The workspace key from unkey.dev
   */
  token: string;

  /**
  * Retry on network errors
  */
  retry?: {
    /**
    * How many attempts should be made
    * The maximum number of requests will be `attempts + 1`
    * `0` means no retries
    *
    * @default 5
    */
    attempts?: number;
    /**
    * Return how many milliseconds to wait until the next attempt is made
    *
    * @default `(retryCount) => Math.round(Math.exp(retryCount) * 10)),`
    */
    backoff?: (retryCount: number) => number;
  };
};

type ApiRequest = {
  path: string[];
  method: "GET" | "POST" | "PUT" | "DELETE";
  body?: unknown;
};

type Result<R> =
  | {
    result: R;
    error?: never;
  }
  | {
    result?: never;
    error: UnkeyError;
  };

export class Unkey {
  public readonly baseUrl: string;
  private readonly token: string;
  public readonly retry: {
    attempts: number;
    backoff: (retryCount: number) => number;
  };

  constructor(opts: UnkeyOptions) {
    this.baseUrl = opts.baseUrl ?? "https://api.unkey.dev";
    /**
     * Even though typescript should prevent this, some people still pass undefined or empty strings
     */
    if (!opts.token) {
      throw new Error("Unkey token must not be empty");
    }
    this.token = opts.token;

    this.retry = {
      attempts: opts.retry?.attempts ?? 5,
      backoff: opts.retry?.backoff ?? ((n) => Math.round(Math.exp(n) * 10))
    }
  }

  private async fetch<TResult>(req: ApiRequest): Promise<Result<TResult>> {
    let res: Response | null = null
    let err: Error | null = null
    for (let i = 0; i <= this.retry.attempts; i++) {
      res = await fetch(`${this.baseUrl}/x${req.path.join("/")}`, {
        method: req.method,
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${this.token}`,
        },
        body: JSON.stringify(req.body),
      }).catch((e: Error) => {
        err = e
        return null // set `res` to `null`
      })
      if (res?.ok) {
        return { result: await res.json() }
      }
      const backoff = this.retry.backoff(i)
      console.debug("attempt %d of %d failed, retrying in %d ms...", i + 1, this.retry.attempts + 1, backoff)
      await new Promise(r => setTimeout(r, backoff))
    }

    if (res) {
      return { error: await res.json() };
    }


    return {
      error: {
        code: "FETCH_ERROR",
        // @ts-ignore I don't understand why `err` is `never`
        message: err?.message ?? "No response",
        docs: "https://developer.mozilla.org/en-US/docs/Web/API/fetch",
        requestId: "N/A"
      }
    }

  }

  public get keys() {
    return {
      create: async (req: {
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
        meta?: unknown;
        /**
         * You can auto expire keys by providing a unix timestamp in milliseconds.
         *
         * Once keys expire they will automatically be deleted and are no longer valid.
         */
        expires?: number;

        /**
         * Unkey comes with per-key ratelimiting out of the box.
         *
         * @see https://docs.unkey.dev/features/ratelimiting
         */
        ratelimit?: {
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

        /**
         * Unkey allows you to set/update usage limits on individual keys
         *
         * @see https://docs.unkey.dev/features/remaining
         */
        remaining?: number;
      }): Promise<Result<{ key: string; keyId: string }>> => {
        return await this.fetch<{ key: string; keyId: string }>({
          path: ["v1", "keys"],
          method: "POST",
          body: req,
        });
      },
      update: async (req: {
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
        meta?: unknown | null;
        /**
         * Update the expiration time, Unix timstamp in milliseconds
         *
         *
         */
        expires?: number | null;

        /**
         * Update the ratelimit
         *
         * @see https://docs.unkey.dev/features/ratelimiting
         */
        ratelimit?: {
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
        } | null;

        /**
         * Update the remaining verifications.
         *
         * @see https://docs.unkey.dev/features/remaining
         */
        remaining?: number | null;
      }): Promise<Result<{ key: string; keyId: string }>> => {
        return await this.fetch<{ key: string; keyId: string }>({
          path: ["v1", "keys", req.keyId],
          method: "PUT",
          body: req,
        });
      },
      verify: async (req: { key: string }): Promise<
        Result<{
          /**
           * Whether or not this key is valid and has passed the ratelimit. If false you should not grant access to whatever the user is requesting
           */
          valid: boolean;

          /**
           * If you have set an ownerId on this key it is returned here. You can use this to clearly authenticate a user in your system.
           */
          ownerId?: string;

          /**
           * This is the meta data you have set when creating the key.
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
          meta?: unknown;
        }>
      > => {
        return await this.fetch<{
          valid: boolean;
          ownerId?: string;
          meta?: unknown;
        }>({
          path: ["v1", "keys", "verify"],
          method: "POST",
          body: req,
        });
      },
      revoke: async (req: { keyId: string }): Promise<Result<void>> => {
        return await this.fetch<void>({
          path: ["v1", "keys", req.keyId],
          method: "DELETE",
        });
      },
    };
  }
  /**
   * Must be authenticated via app token
   */
  public get _internal() {
    return {
      createRootKey: async (req: {
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
      }): Promise<Result<{ key: string; keyId: string }>> => {
        return await this.fetch<{ key: string; keyId: string }>({
          path: ["v1", "internal", "rootkeys"],
          method: "POST",
          body: req,
        });
      },
    };
  }
}

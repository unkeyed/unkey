export type UnkeyOptions = {
  /**
   * @default https://api.unkey.dev
   */
  baseUrl?: string;

  /**
   * The workspace key from unkey.dev
   */
  token: string;
};

type ApiRequest = {
  path: string[];
  method: "GET" | "POST" | "PUT" | "DELETE";
  body?: unknown;
};

export class Unkey {
  public readonly baseUrl: string;
  private readonly token: string;

  constructor(opts: UnkeyOptions) {
    this.baseUrl = opts.baseUrl ?? "https://api.unkey.dev";
    /**
     * Even though typescript should prevent this, some people still pass undefined or empty strings
     */
    if (!opts.token) {
      throw new Error("Unkey token must not be empty");
    }
    this.token = opts.token;
  }

  private async fetch<TResult>(req: ApiRequest): Promise<TResult> {
    const url = `${this.baseUrl}/${req.path.join("/")}`;
    console.log("fetching", url);
    const res = await fetch(url, {
      method: req.method,
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${this.token}`,
      },
      body: JSON.stringify(req.body),
    });
    if (!res.ok) {
      throw new Error(await res.text());
    }
    return await res.json();
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
      }): Promise<{ key: string; keyId: string }> => {
        return await this.fetch<{ key: string; keyId: string }>({
          path: ["v1", "keys"],
          method: "POST",
          body: req,
        });
      },
      verify: async (req: { key: string }): Promise<{
        /**
         * Whether or not this key is valid and has passed the ratelimit. If false you should not grant access to whatever the user is requesting
         */
        valid: boolean;

        /**
         * If you have set an ownerId on this key it is returned here. You can use this to clearly authenticate a user in your system.
         */
        ownerId?: string;

        meta?: unknown;

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
      }> => {
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
      revoke: async (req: { keyId: string }): Promise<void> => {
        await this.fetch<{
          valid: boolean;
          ownerId?: string;
          meta?: unknown;
        }>({
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
      }): Promise<{ key: string; keyId: string }> => {
        return await this.fetch<{ key: string; keyId: string }>({
          path: ["v1", "internal", "rootkeys"],
          method: "POST",
          body: req,
        });
      },
    };
  }
}

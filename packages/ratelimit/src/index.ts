import { Duration } from "./duration";

type Limit = {
  /**
   * How many requests may pass in the given duration
   */
  limit: number;

  /**
   * Either a type string literal or milliseconds
   */
  duration: Duration | number;
};

export type RatelimitResponse = {
  /**
   * Whether the request may pass(true) or exceeded the limit(false)
   */
  success: boolean;
  /**
   * Maximum number of requests allowed within a window.
   */
  limit: number;
  /**
   * How many requests the user has left within the current window.
   */
  remaining: number;
  /**
   * Unix timestamp in milliseconds when the limits are reset.
   */
  reset: number;
};

export type RatelimitConfig = Limit & {
  rootKey: string;

  /**
   * Something to differentiate between different services. You can filter on this in the analytics
   * pages.
   */
  namespace: string;

  cache: Map<string, number>;

  timeout?: number;

  consistency?: "fast" | "edge" | "strict";
};

export interface Ratelimiter {
  limit: (
    identifier: string,
    opts?: {
      /**
       * Separate requests into groups, groups are combined with your identifier and can be filtered
       * and searched later.
       *
       * @example `group: "send.email"` -> `send.email_${userId}`
       */
      group?: string;

      /**
       * Expensive requests may use up more resources. You can specify a cost to the request and
       * we'll deduct this many tokens in the current window. If there are not enough tokens left,
       * the request is denied.
       *
       * @example
       *
       * 1. You have a limit of 10 requests per second you already used 4 of them in the current
       * window.
       *
       * 2. Now a new request comes in with a higher cost:
       * ```ts
       * const res = await rl.limit("identifier", { cost: 4 })
       * ```
       *
       * 3. The request passes and the current limit is now at `8`
       *
       * 4. The same request happens again, but would not be rejected, because it would exceed the
       * limit in the current window: `8 + 4 > 10`
       *
       *
       * @default 1
       */
      cost?: number;

      /**
       * Override the default limit.
       *
       * This takes precedence over the limit defined in the constructor as well as any limits defined
       * for this identifier in Unkey.
       */
      limit?: Limit;

      /**
       * TODO: This does nothing right now
       */
      async?: boolean;

      /**
       * Record arbitrary data about this request. This does not affect the limit itself but can help
       * you debug later.
       */
      meta?: Record<string, string | number | boolean | null>;

      /**
       * Specify which resources this request would access and we'll create a papertrail for you.
       *
       * @see https://unkey.dev/app/audit
       */
      resources?: {
        type: string;
        id: string;
        name?: string;
        meta?: Record<string, string | number | boolean | null>;
      }[];
    },
  ) => Promise<RatelimitResponse>;

  // retrieveLimit: (identifier: string) => Promise<TODO>;
  // reset: (identifier: string) => Promise<void>;
}

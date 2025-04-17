import type { Duration } from "./duration";

export type Limit = {
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

  /**
   * The override id for the request that was used to override the limit.
   */
  overrideId?: string;
};

export type LimitOptions = {
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
   * Do not wait for a response from the origin. Faster but less accurate.
   */
  // async?: boolean;

  /**
   * Record arbitrary data about this request. This does not affect the limit itself but can help
   * you debug later.
   */
  // meta?: Record<string, string | number | boolean | null>;

  /**
   * Specify which resources this request would access and we'll create a papertrail for you.
   *
   * @see https://unkey.dev/app/audit
   */
  // resources?: {
  //   type: string;
  //   id: string;
  //   name?: string;
  //   meta?: Record<string, string | number | boolean | null>;
  // }[];
};

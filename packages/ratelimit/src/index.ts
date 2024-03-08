import { Duration } from "./duration";

type TODO = any;

type Limit = {
  tokens: number;
  duration: Duration | number;
};

type RatelimitConfig = Limit & {
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
       * Separate requests into buckets, buckets are combined with your identifier
       *
       * @example `send.email` -> `send.email_${userId}`
       */
      bucket?: string;

      /**
       * Expensive requests may use up more tokens. You can specify a cost to the request here and
       * we'll deduct this many tokens in the current window. If there are not enough tokens left,
       * the request is denied.
       *
       * @default 1
       */
      tokens?: number;

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

      meta?: Record<string, string | number | boolean | null>;

      resources?: {
        type: string;
        id: string;
        name?: string;
        meta?: Record<string, string | number | boolean | null>;
      }[];
    },
  ) => Promise<TODO>;

  retrieveLimit: (identifier: string) => Promise<TODO>;
  reset: (identifier: string) => Promise<void>;
}

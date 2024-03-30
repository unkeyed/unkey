import type { LimitOptions, RatelimitResponse } from "./types";

export interface Ratelimiter {
  limit: (identifier: string, opts?: LimitOptions) => Promise<RatelimitResponse>;

  // retrieveLimit: (identifier: string) => Promise<TODO>;
  // reset: (identifier: string) => Promise<void>;
}

import type { BaseError, Result } from "@unkey/error";
import type { Logger } from "@unkey/worker-logging";

export type RetryOptions = {
  logger: Logger;
  attempts: number;
  backoff: (n: number) => number;
};

export class Retry {
  private readonly logger: Logger;
  private readonly attempts: number;
  private readonly backoff: (n: number) => number;

  constructor(opts: RetryOptions) {
    this.logger = opts.logger;
    this.attempts = opts.attempts;
    this.backoff = opts.backoff;
  }

  public async retry<T, E extends BaseError>(
    fn: () => Promise<Result<T, E>>,
  ): Promise<Result<T, E>> {
    let result: Result<T, E>;
    for (let i = 0; i < this.attempts; i++) {
      result = await fn();
      if (!result.err) {
        return result;
      }

      const backoff = this.backoff(i);
      this.logger.warn("attempt failed", {
        function: fn.name,
        attempt: i,
        backoff,
      });
      await new Promise((r) => setTimeout(r, backoff));
    }
    // biome-ignore lint/style/noNonNullAssertion: Safe to leave
    return result!;
  }
}

export function retry<T>(
  attempts: number,
  fn: (attempt: number) => T,
  log?: (attempt: number, e: Error) => void,
): T {
  let err: Error | undefined = undefined;
  for (let i = 1; i <= attempts; i++) {
    try {
      return fn(i);
    } catch (e) {
      if (log) {
        log(i, e as Error);
      } else {
        console.warn(e);
      }
      err = e as Error;
    }
  }
  throw err;
}

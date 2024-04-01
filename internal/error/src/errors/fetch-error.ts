import { BaseError } from "./base";

/**
 * Fetch Errors
 */
export class FetchError extends BaseError<{
  url: string;
  method: string;
  [more: string]: unknown;
}> {
  public readonly retry: boolean;

  constructor(
    message: string,
    opts: {
      retry: boolean;
      cause?: BaseError;
      context?: {
        url: string;
        method: string;
        [more: string]: unknown;
      };
    },
  ) {
    super(message, {
      ...opts,
      id: FetchError.name,
    });
    this.retry = opts.retry;
  }
}

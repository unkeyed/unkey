import { BaseError } from "./base";

/**
 * Env Errors indicate an environment variable was not configured properly
 */
export class EnvError extends BaseError<{
  name: string;
}> {
  public readonly retry: boolean;

  constructor(
    message: string,
    opts: {
      context?: {
        name: string;
      };
    },
  ) {
    super(message, {
      ...opts,
      id: EnvError.name,
    });
    this.retry = false;
  }
}

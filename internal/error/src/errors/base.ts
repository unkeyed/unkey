export type ErrorType = "FetchError" | "SchemaError" | "CacheError" | "VercelCodeExchangeError";

export type ErrorSource = "UPSTREAM" | "DOWNSTREAM" | "INTERNAL" | null;
export type ErrorContext = Record<string, unknown>;

export abstract class BaseError<TContext extends ErrorContext = ErrorContext> extends Error {
  public abstract readonly type: ErrorType;
  public abstract readonly retry: boolean;
  public abstract readonly source: ErrorSource;
  public readonly cause: BaseError | undefined;
  public readonly context: TContext | undefined;

  constructor(
    message: string,
    opts?: {
      cause?: BaseError;
      context?: TContext;
    },
  ) {
    super(message);
    this.cause = opts?.cause;
    this.context = opts?.context;
  }

  public toString(): string {
    return `${this.type}: ${this.message} - ${JSON.stringify(
      this.context,
    )} - caused by ${this.cause?.toString()}`;
  }
}

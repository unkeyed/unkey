import { BaseError } from "@unkey/error";
export class InsertError extends BaseError {
  public readonly retry = true;
  public readonly name = InsertError.name;
  constructor(message: string) {
    super({
      message,
    });
  }
}
export class QueryError extends BaseError<{ query: string }> {
  public readonly retry = true;
  public readonly name = QueryError.name;
  constructor(message: string, context: { query: string }) {
    super({
      message,
      context,
    });
  }
}

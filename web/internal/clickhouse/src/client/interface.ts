import type { Result } from "@unkey/error";
import type { z } from "zod";
import type { InsertError, QueryError } from "./error";
export interface Querier {
  query<TIn extends z.ZodType<unknown>, TOut extends z.ZodType<unknown>>(req: {
    // The SQL query to run.
    // Use {paramName: Type} to define parameters
    // Example: `SELECT * FROM table WHERE id = {id: String}`
    query: string;
    // The schema of the parameters
    // Example: z.object({ id: z.string() })
    params?: TIn;
    // The schema of the output of each row
    // Example: z.object({ id: z.string() })
    schema: TOut;
  }): (params: z.input<TIn>) => Promise<Result<z.output<TOut>[], QueryError>>;
}

export interface Inserter {
  insert<TSchema extends z.ZodType<unknown>>(req: {
    table: string;
    schema: TSchema;
  }): (
    events: z.input<TSchema> | z.input<TSchema>[],
  ) => Promise<Result<{ executed: boolean; query_id: string }, InsertError>>;
}

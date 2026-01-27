import { Err, Ok, type Result } from "@unkey/error";
import type { z } from "zod";
import { InsertError, type QueryError } from "./error";
import type { Inserter, Querier } from "./interface";
export class Noop implements Querier, Inserter {
  public query<TIn extends z.ZodType<unknown>, TOut extends z.ZodType<unknown>>(req: {
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
  }): (params: z.input<TIn>) => Promise<Result<z.output<TOut>[], QueryError>> {
    return async (params: z.input<TIn>): Promise<Result<z.output<TOut>[], QueryError>> => {
      req.params?.safeParse(params);
      return Ok([]);
    };
  }

  public insert<TSchema extends z.ZodType<unknown>>(req: {
    table: string;
    schema: TSchema;
  }): (
    events: z.input<TSchema> | z.input<TSchema>[],
  ) => Promise<Result<{ executed: boolean; query_id: string }, InsertError>> {
    return async (events: z.input<TSchema> | z.input<TSchema>[]) => {
      const v = Array.isArray(events)
        ? req.schema.array().safeParse(events)
        : req.schema.safeParse(events);
      if (!v.success) {
        return Err(new InsertError(v.error.message));
      }

      return Ok({ executed: true, query_id: "noop" });
    };
  }
}

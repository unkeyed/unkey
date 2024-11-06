import type { z } from "zod";
import type { Inserter, Querier } from "./interface";
export class Noop implements Querier, Inserter {
  public query<TIn extends z.ZodSchema<any>, TOut extends z.ZodSchema<any>>(req: {
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
  }): (params: z.input<TIn>) => Promise<z.output<TOut>[]> {
    return async (params: z.input<TIn>): Promise<z.output<TOut>[]> => {
      req.params?.safeParse(params);
      return [];
    };
  }

  public insert<TSchema extends z.ZodSchema<any>>(req: {
    table: string;
    schema: TSchema;
  }): (
    events: z.input<TSchema> | z.input<TSchema>[],
  ) => Promise<{ executed: boolean; query_id: string }> {
    return async (events: z.input<TSchema> | z.input<TSchema>[]) => {
      const v = Array.isArray(events)
        ? req.schema.array().safeParse(events)
        : req.schema.safeParse(events);
      if (!v.success) {
        throw new Error(v.error.message);
      }

      return { executed: true, query_id: "noop" };
    };
  }
}

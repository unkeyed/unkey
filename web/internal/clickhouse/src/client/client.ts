import { type ClickHouseClient, createClient } from "@clickhouse/client-web";
import { Err, Ok, type Result } from "@unkey/error";
import { z } from "zod";
import { InsertError, QueryError } from "./error";
import type { Inserter, Querier } from "./interface";
export type Config = {
  url: string;
  request_timeout?: number;
};

export class Client implements Querier, Inserter {
  private readonly client: ClickHouseClient;

  constructor(config: Config) {
    this.client = createClient({
      url: config.url,
      // config time out otherwise set it to 30 seconds
      request_timeout: config.request_timeout ?? 30000,
      clickhouse_settings: {
        output_format_json_quote_64bit_integers: 0,
        output_format_json_quote_64bit_floats: 0,
      },
    });
  }

  // biome-ignore lint/suspicious/noExplicitAny: Safe to leave
  public query<TIn extends z.ZodSchema<any>, TOut extends z.ZodSchema<unknown>>(req: {
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
      const validParams = req.params?.safeParse(params);
      if (validParams?.error) {
        return Err(new QueryError(`Bad params: ${validParams.error.message}`, { query: "" }));
      }
      let unparsedRows: Array<TOut> = [];
      try {
        const res = await this.client.query({
          query: req.query,
          query_params: validParams?.data, // Default to empty object
          format: "JSONEachRow",
        });
        unparsedRows = await res.json();
      } catch (err) {
        const message = err instanceof Error ? err.message : JSON.stringify(err);
        console.error(err);

        return Err(new QueryError(`Unable to query clickhouse: ${message}`, { query: req.query }));
      }
      const parsed = z.array(req.schema).safeParse(unparsedRows);
      if (parsed.error) {
        return Err(new QueryError(`Malformed data: ${parsed.error.message}`, { query: req.query }));
      }
      return Ok(parsed.data);
    };
  }

  public insert<TSchema extends z.ZodSchema<unknown>>(req: {
    table: string;
    schema: TSchema;
  }): (
    events: z.input<TSchema> | z.input<TSchema>[],
  ) => Promise<Result<{ executed: boolean; query_id: string }, InsertError>> {
    return async (events: z.input<TSchema> | z.input<TSchema>[]) => {
      let validatedEvents: z.output<TSchema> | z.output<TSchema>[] | undefined = undefined;
      const v = Array.isArray(events)
        ? req.schema.array().safeParse(events)
        : req.schema.safeParse(events);
      if (!v.success) {
        return Err(new InsertError(v.error.message));
      }
      validatedEvents = v.data;

      return this.retry(() =>
        this.client
          .insert({
            table: req.table,
            format: "JSONEachRow",
            values: Array.isArray(validatedEvents) ? validatedEvents : [validatedEvents],
          })
          .then((res) => Ok(res))
          .catch((err) => Err(new InsertError(err.message))),
      );
    };
  }

  private async retry<T>(fn: (attempt: number) => Promise<T>): Promise<T> {
    let err: Error | undefined = undefined;
    for (let i = 1; i <= 3; i++) {
      try {
        return fn(i);
      } catch (e) {
        console.warn(e);
        err = e as Error;
      }
    }
    throw err;
  }
}

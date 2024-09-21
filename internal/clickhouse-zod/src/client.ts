import { type ClickHouseClient, createClient } from "@clickhouse/client-web";
import type { z } from "zod";

export type Config =
  | {
      url: string;
      noop?: never;
    }
  | {
      url?: never;
      noop: true;
    };

export class Clickhouse {
  private readonly client: ClickHouseClient;
  private readonly noop: boolean;

  constructor(config: Config) {
    this.noop = config.noop || false;
    this.client = createClient({
      url: config.url,
      clickhouse_settings: {
        async_insert: 1,
        wait_for_async_insert: 1,
        output_format_json_quote_64bit_integers: 0,
        output_format_json_quote_64bit_floats: 0,
      },
    });
  }

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
      if (this.noop) {
        return [];
      }

      const res = await this.client.query({
        query: req.query,
        query_params: params,
        format: "JSONEachRow",
      });
      const rows = await res.json();
      return rows.map((row) => req.schema.parse(row));
    };
  }

  public insert<TSchema extends z.ZodSchema<any>>(req: {
    table: string;
    schema: TSchema;
  }): (
    events: z.input<TSchema> | z.input<TSchema>[],
  ) => Promise<{ executed: boolean; query_id: string }> {
    return async (events: z.input<TSchema> | z.input<TSchema>[]) => {
      let validatedEvents: z.output<TSchema> | z.output<TSchema>[] | undefined = undefined;
      if (req.schema) {
        const v = Array.isArray(events)
          ? req.schema.array().safeParse(events)
          : req.schema.safeParse(events);
        if (!v.success) {
          throw new Error(v.error.message);
        }
        validatedEvents = v.data;
      }

      if (this.noop) {
        return { executed: true, query_id: "noop" };
      }

      return await this.client.insert({
        table: req.table,
        format: "JSONEachRow",
        values: Array.isArray(validatedEvents) ? validatedEvents : [validatedEvents],
      });
    };
  }
}

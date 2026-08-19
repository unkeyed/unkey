import { Ok, type Result } from "@unkey/error";
import type { z } from "zod";
import type { QueryError } from "./client/error";
import type { Querier } from "./client/interface";

export class CapturingQuerier implements Querier {
  public readonly queries: string[] = [];
  public readonly params: unknown[] = [];

  public query<TIn extends z.ZodType<unknown>, TOut extends z.ZodType<unknown>>(req: {
    query: string;
    params?: TIn;
    schema: TOut;
  }): (params: z.input<TIn>) => Promise<Result<z.output<TOut>[], QueryError>> {
    this.queries.push(req.query);
    return async (params) => {
      this.params.push(params);
      return Ok([]);
    };
  }
}

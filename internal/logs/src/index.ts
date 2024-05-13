import { metricSchema } from "@unkey/metrics";
import { z } from "zod";

export const logContext = z.object({
  requestId: z.string(),
});

export const logSchema = z.discriminatedUnion("type", [
  z.object({
    type: z.literal("log"),
    level: z.enum(["debug", "info", "warn", "error"]),
    time: z.number(),
    message: z.string(),
    context: z.record(z.any()),
  }),
  z.object({
    type: z.literal("metric"),
    time: z.number(),
    metric: metricSchema,
  }),
]);

export class Log<TLog extends z.infer<typeof logSchema>> {
  public readonly ctx: z.infer<typeof logContext>;
  public readonly log: TLog;

  constructor(ctx: z.infer<typeof logContext>, log: TLog) {
    this.ctx = ctx;
    this.log = log;
  }

  public toString(): string {
    return JSON.stringify({ ctx: this.ctx, ...this.log });
  }
}

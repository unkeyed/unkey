import { metricSchema } from "@unkey/metrics";
import { z } from "zod";

export const logContext = z.object({
  requestId: z.string(),
});

export const logSchema = z.discriminatedUnion("type", [
  z.object({
    type: z.literal("log"),
    level: z.enum(["debug", "info", "warn", "error", "fatal"]),
    requestId: z.string(),
    time: z.number(),
    message: z.string(),
    context: z.record(z.any()),
  }),
  z.object({
    type: z.literal("metric"),
    requestId: z.string(),
    time: z.number(),
    metric: metricSchema,
  }),
]);

export class Log<TLog extends z.infer<typeof logSchema>> {
  public readonly log: TLog;

  constructor(log: TLog) {
    this.log = log;
  }

  public toString(): string {
    return JSON.stringify(this.log);
  }
}

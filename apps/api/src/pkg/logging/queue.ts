import { Queue } from "@cloudflare/workers-types";
import { BufferQueue } from "@unkey/zod-queue";
import { z } from "zod";
import { ConsoleLogger } from "./console";
import { Fields, Logger } from "./interface";

const schema = z
  .object({
    message: z.string(),
    level: z.enum(["debug", "info", "warn", "error"]),
  })
  .passthrough();

export class QueueLogger implements Logger {
  private readonly consoleLogger: Logger;
  private readonly queue: BufferQueue<typeof schema>;

  constructor(opts: {
    queue: Queue<z.infer<typeof schema>[]>;
  }) {
    this.consoleLogger = new ConsoleLogger();
    this.queue = new BufferQueue({
      queue: opts.queue,
      queueSendOptions: { contentType: "json" },
      schema,
    });
  }

  public debug(message: string, fields?: Fields): void {
    this.consoleLogger.debug(message, fields);
    this.queue.buffer({
      level: "debug",
      message,
      ...fields,
    });
  }
  public info(message: string, fields?: Fields): void {
    this.consoleLogger.info(message, fields);
    this.queue.buffer({
      level: "info",
      message,
      ...fields,
    });
  }
  public warn(message: string, fields?: Fields): void {
    this.consoleLogger.warn(message, fields);
    this.queue.buffer({
      level: "warn",
      message,
      ...fields,
    });
  }
  public error(message: string, fields?: Fields): void {
    this.consoleLogger.error(message, fields);
    this.queue.buffer({
      level: "error",
      message,
      ...fields,
    });
  }
  /**
   * Call this at the end of the request handler with .waitUntil()
   */

  public async flush(): Promise<void> {
    await this.queue.flush();
  }
}

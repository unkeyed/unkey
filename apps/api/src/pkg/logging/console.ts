import { Log } from "@unkey/logs";
import type { Fields, Logger } from "./interface";

export class ConsoleLogger implements Logger {
  private readonly defaultFields?: Fields;

  constructor(opts?: { defaultFields?: Fields }) {
    this.defaultFields = opts?.defaultFields;
  }

  private marshal(
    level: "debug" | "info" | "warn" | "error",
    message: string,
    fields?: Fields,
  ): string {
    return new Log({
      type: "log",
      time: Date.now(),
      level,
      message,
      context: { ...this.defaultFields, ...fields },
    }).toString();
  }

  public debug(message: string, fields?: Fields): void {
    console.debug(this.marshal("debug", message, fields));
  }
  public info(message: string, fields?: Fields): void {
    console.info(this.marshal("info", message, fields));
  }
  public warn(message: string, fields?: Fields): void {
    console.warn(this.marshal("warn", message, fields));
  }
  public error(message: string, fields?: Fields): void {
    console.error(this.marshal("error", message, fields));
  }
}

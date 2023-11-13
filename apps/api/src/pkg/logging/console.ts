import type { Fields, Logger } from "./interface";

export class ConsoleLogger implements Logger {
  private readonly defaultFields?: Fields;

  constructor(opts?: { defaultFields?: Fields }) {
    this.defaultFields = opts?.defaultFields;
  }

  private buildMessage(
    message: string,
    fields?: Fields,
  ): [message?: any, ...optionalParams: any[]] {
    const f = { ...this.defaultFields, ...fields };
    if (Object.keys(f).length > 0) {
      return [message, JSON.stringify(f)];
    }
    return [message];
  }

  public debug(message: string, fields?: Fields): void {
    console.debug(...this.buildMessage(message, fields));
  }
  public info(message: string, fields?: Fields): void {
    console.info(...this.buildMessage(message, fields));
  }
  public warn(message: string, fields?: Fields): void {
    console.warn(...this.buildMessage(message, fields));
  }
  public error(message: string, fields?: Fields): void {
    console.error(...this.buildMessage(message, fields));
  }
  public flush() {
    return Promise.resolve();
  }
}

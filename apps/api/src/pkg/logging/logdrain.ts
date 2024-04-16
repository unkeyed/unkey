import type { Fields, Logger } from "./interface";

export class LogdrainLogger implements Logger {
  private readonly defaultFields?: Fields;

  constructor(opts?: { defaultFields?: Fields }) {
    this.defaultFields = opts?.defaultFields;
  }

  private marshal(message: string, fields?: Fields): string {
    return JSON.stringify({ message, ...this.defaultFields, ...fields });
  }

  public debug(message: string, fields?: Fields): void {
    console.debug(this.marshal(message, fields));
  }
  public info(message: string, fields?: Fields): void {
    console.info(this.marshal(message, fields));
  }
  public warn(message: string, fields?: Fields): void {
    console.warn(this.marshal(message, fields));
  }
  public error(message: string, fields?: Fields): void {
    console.error(this.marshal(message, fields));
  }
  public flush() {
    return Promise.resolve();
  }
}

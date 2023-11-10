import { Fields, Logger } from "./interface";

import { ConsoleLogger } from "./console";

export class AxiomLogger implements Logger {
  private readonly consoleLogger: Logger;
  private readonly axiomDataset: string;
  private readonly axiomToken: string;
  private readonly defaultFields: Fields;
  private buffer: unknown[] = [];

  /**
   * @param opts.axiomToken The token to use to authenticate with axiom
   * @param opts.defaultFields Any additional defaultFields to add to the metrics by default
   */
  constructor(opts: { axiomToken: string; defaultFields?: Fields }) {
    this.consoleLogger = new ConsoleLogger();
    this.axiomDataset = "cf_api_logs";
    this.axiomToken = opts.axiomToken;
    this.defaultFields = opts.defaultFields ?? {};
  }

  public debug(message: string, fields?: Fields): void {
    this.consoleLogger.debug(message, fields);
    this.buffer.push({
      level: "debug",
      _time: Date.now(),
      message,
      ...this.defaultFields,
      ...fields,
    });
  }
  public info(message: string, fields?: Fields): void {
    this.consoleLogger.info(message, fields);
    this.buffer.push({
      level: "info",
      _time: Date.now(),
      message,
      ...this.defaultFields,
      ...fields,
    });
  }
  public warn(message: string, fields?: Fields): void {
    this.consoleLogger.warn(message, fields);
    this.buffer.push({
      level: "warn",
      _time: Date.now(),
      message,
      ...this.defaultFields,
      ...fields,
    });
  }
  public error(message: string, fields?: Fields): void {
    this.consoleLogger.error(message, fields);
    this.buffer.push({
      level: "error",
      _time: Date.now(),
      message,
      ...this.defaultFields,
      ...fields,
    });
  }

  /**
   * flush sends the metrics to axiom
   *
   * Call this at the end of the request handler with .waitUntil()
   */
  public async flush(): Promise<void> {
    const copy = this.buffer.slice();
    this.buffer = [];
    await fetch(`https://api.axiom.co/v1/datasets/${this.axiomDataset}/ingest`, {
      method: "POST",
      body: JSON.stringify(copy),
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${this.axiomToken} `,
      },
    }).catch((err) => {
      console.error("unable to ingest to axiom", err);
    });
  }
}

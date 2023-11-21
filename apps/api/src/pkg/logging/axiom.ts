import { Axiom } from "@axiomhq/js";
import { Fields, Logger } from "./interface";

import { Env } from "../env";
import { ConsoleLogger } from "./console";

export class AxiomLogger implements Logger {
  private readonly consoleLogger: Logger;
  private readonly axiomDataset: string;
  private readonly ax: Axiom;
  private readonly defaultFields: Fields;

  /**
   * @param opts.axiomToken The token to use to authenticate with axiom
   * @param opts.defaultFields Any additional defaultFields to add to the metrics by default
   */
  constructor(opts: {
    axiomToken: string;
    defaultFields?: Fields;
    environment: Env["ENVIRONMENT"];
  }) {
    this.consoleLogger = new ConsoleLogger();
    this.axiomDataset = `cf_api_logs_${opts.environment}`;
    this.ax = new Axiom({
      token: opts.axiomToken,
    });

    this.defaultFields = opts.defaultFields ?? {};
  }

  public debug(message: string, fields?: Fields): void {
    this.consoleLogger.debug(message, fields);
    this.ax.ingest(this.axiomDataset, [
      {
        level: "debug",
        _time: Date.now(),
        message,
        ...this.defaultFields,
        ...fields,
      },
    ]);
  }
  public info(message: string, fields?: Fields): void {
    this.consoleLogger.info(message, fields);
    this.ax.ingest(this.axiomDataset, [
      {
        level: "info",
        _time: Date.now(),
        message,
        ...this.defaultFields,
        ...fields,
      },
    ]);
  }
  public warn(message: string, fields?: Fields): void {
    this.consoleLogger.warn(message, fields);
    this.ax.ingest(this.axiomDataset, [
      {
        level: "warn",
        _time: Date.now(),
        message,
        ...this.defaultFields,
        ...fields,
      },
    ]);
  }
  public error(message: string, fields?: Fields): void {
    this.consoleLogger.error(message, fields);
    this.ax.ingest(this.axiomDataset, [
      {
        level: "error",
        _time: Date.now(),
        message,
        ...this.defaultFields,
        ...fields,
      },
    ]);
  }

  /**
   * flush sends the metrics to axiom
   *
   * Call this at the end of the request handler with .waitUntil()
   */
  public async flush(): Promise<void> {
    await this.ax.flush().catch((err) => {
      this.consoleLogger.error("unable to flush logs to axiom", err);
    });
  }
}

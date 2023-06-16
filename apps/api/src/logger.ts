import { Client as Axiom } from "@axiomhq/axiom-node";
import * as tslog from "tslog";

export class Logger {
  private logger: tslog.Logger<tslog.ILogObj>;
  private meta: Record<string, unknown> = {};
  constructor(meta?: Record<string, unknown>) {
    this.logger = new tslog.Logger({
      type: process.env.NODE_ENV === "production" ? "json" : "pretty",
      minLevel: 3, // info and above
    });
    if (meta) {
      this.meta = meta;
    }

    const axiomToken = process.env.AXIOM_TOKEN;
    const axiomOrgId = process.env.AXIOM_ORG_ID;
    if (axiomToken && axiomOrgId) {
      const axiom = new Axiom({
        token: axiomToken,
        orgId: axiomOrgId,
      });

      this.logger.attachTransport((logObj) => {
        axiom.ingestEvents("api", [logObj]).catch((err) => {
          console.error(`Failed to send log to axiom: ${err.message}`);
        });
      });
    }
  }

  public with(fields: Record<string, unknown>) {
    return new Logger({ ...this.meta, ...fields });
  }

  public debug(message: string, fields?: Record<string, unknown>) {
    this.logger.debug({ message, ...fields, ...this.meta });
  }
  public info(message: string, fields?: Record<string, unknown>) {
    this.logger.info({ message, ...fields, ...this.meta });
  }
  public warn(message: string, fields?: Record<string, unknown>) {
    this.logger.warn({ message, ...fields, ...this.meta });
  }
  public error(message: string, fields?: Record<string, unknown>) {
    this.logger.error({ message, ...fields, ...this.meta });
  }
}

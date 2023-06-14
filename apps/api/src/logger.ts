import { Client as Axiom } from "@axiomhq/axiom-node";
import * as tslog from "tslog";

export class Logger {
  private logger: tslog.Logger<tslog.ILogObj>;

  constructor() {
    this.logger = new tslog.Logger({
      type: process.env.NODE_ENV === "production" ? "json" : "pretty",
      minLevel: 3, // info and above
    });

    const axiomToken = process.env.AXIOM_TOKEN;
    const axiomOrgId = process.env.AXIOM_ORG_ID;
    if (axiomToken && axiomOrgId) {
      const axiom = new Axiom({
        token: axiomToken,
        orgId: axiomOrgId,
      });

      this.logger.attachTransport((logObj) => {
        axiom.ingestEvents("scheduler", [logObj]).catch((err) => {
          console.error(`Failed to send log to axiom: ${err.message}`);
        });
      });
    }
  }

  public debug(message: string, fields?: Record<string, unknown>) {
    this.logger.debug({ message, ...fields });
  }
  public info(message: string, fields?: Record<string, unknown>) {
    this.logger.info({ message, ...fields });
  }
  public warn(message: string, fields?: Record<string, unknown>) {
    this.logger.warn({ message, ...fields });
  }
  public error(message: string, fields?: Record<string, unknown>) {
    this.logger.error({ message, ...fields });
  }
}

import { Database, createConnection, eq, schema, sql } from "@/pkg/db";
import type { Key } from "@unkey/db";
import { Env } from "../env";
import { ConsoleLogger, Logger } from "../logging";
import { AxiomLogger } from "../logging/axiom";
import { limitRequestSchema, revalidateRequestSchema } from "./interface";

export class DurableObjectUsagelimiter {
  private readonly state: DurableObjectState;
  private readonly db: Database;
  private key: Key | undefined = undefined;
  private readonly logger: Logger;
  constructor(state: DurableObjectState, env: Env["Bindings"]) {
    this.state = state;
    this.db = createConnection({
      host: env.DATABASE_HOST,
      password: env.DATABASE_PASSWORD,
      username: env.DATABASE_USERNAME,
    });

    const defaultFields = {
      durableObjectId: state.id.toString(),
      durableObjectClass: state.constructor.name,
    };
    this.logger = env.AXIOM_TOKEN
      ? new AxiomLogger({
          axiomToken: env.AXIOM_TOKEN,
          environment: env.ENVIRONMENT,
          defaultFields,
        })
      : new ConsoleLogger({ defaultFields });
  }

  async fetch(request: Request) {
    const url = new URL(request.url);
    switch (url.pathname) {
      case "/revalidate": {
        const req = revalidateRequestSchema.parse(await request.json());
        if (!this.key) {
          this.logger.info("Fetching key from origin", { id: req.keyId });
          this.key = await this.db.query.keys.findFirst({
            where: (table, { and, eq, isNull }) =>
              and(eq(table.id, req.keyId), isNull(table.deletedAt)),
          });
        }

        this.key = await this.db.query.keys.findFirst({
          where: (table, { and, eq, isNull }) =>
            and(eq(table.id, req.keyId), isNull(table.deletedAt)),
        });
        return Response.json({});
      }
      case "/limit": {
        const req = limitRequestSchema.parse(await request.json());
        if (!this.key) {
          this.logger.info("Fetching key from origin", { id: req.keyId });
          this.key = await this.db.query.keys.findFirst({
            where: (table, { and, eq, isNull }) =>
              and(eq(table.id, req.keyId), isNull(table.deletedAt)),
          });
        }
        if (!this.key) {
          this.logger.error("key not found", { keyId: req.keyId });
          return Response.json({
            valid: false,
          });
        }

        if (this.key.remainingRequests === null) {
          this.logger.warn("key does not have remaining requests enabled", { key: this.key });
          return Response.json({
            valid: true,
          });
        }

        if (this.key.remainingRequests <= 0) {
          return Response.json({
            valid: false,
            remaining: 0,
          });
        }

        this.key.remainingRequests = Math.max(0, this.key.remainingRequests - 1);

        this.state.waitUntil(
          this.db
            .update(schema.keys)
            .set({ remainingRequests: sql`${schema.keys.remainingRequests}-1` })
            .where(eq(schema.keys.id, this.key.id)),
        );

        return Response.json({
          valid: true,
          remaining: this.key.remainingRequests,
        });
      }
    }
  }
}

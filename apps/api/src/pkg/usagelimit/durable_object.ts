import { type Database, type Key, and, createConnection, eq, gt, schema, sql } from "@/pkg/db";
import { Env } from "../env";
import { ConsoleLogger, Logger } from "../logging";
import { AxiomLogger } from "../logging/axiom";
import { limitRequestSchema, revalidateRequestSchema } from "./interface";

export class DurableObjectUsagelimiter {
  private readonly state: DurableObjectState;
  private readonly db: Database;
  private lastRevalidate = 0;
  private key: Key | undefined = undefined;
  private readonly logger: Logger;
  constructor(state: DurableObjectState, env: Env) {
    this.state = state;
    this.db = createConnection({
      host: env.DATABASE_HOST,
      password: env.DATABASE_PASSWORD,
      username: env.DATABASE_USERNAME,
    });

    const defaultFields = {
      durableObjectId: state.id.toString(),
      durableObjectClass: "DurableObjectUsagelimiter",
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

        this.key = await this.db.query.keys.findFirst({
          where: (table, { and, eq, isNull }) =>
            and(eq(table.id, req.keyId), isNull(table.deletedAt)),
        });
        this.lastRevalidate = Date.now();
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
          this.lastRevalidate = Date.now();
        }

        if (!this.key) {
          this.logger.error("key not found", { keyId: req.keyId });
          return Response.json({
            valid: false,
          });
        }

        if (this.key.remaining === null) {
          this.logger.warn("key does not have remaining requests enabled", { key: this.key });
          return Response.json({
            valid: true,
          });
        }

        if (this.key.remaining <= 0) {
          return Response.json({
            valid: false,
            remaining: 0,
          });
        }

        this.key.remaining = Math.max(0, this.key.remaining - 1);

        this.state.waitUntil(
          this.db
            .update(schema.keys)
            .set({ remaining: sql`${schema.keys.remaining}-1` })
            .where(
              and(
                eq(schema.keys.id, this.key.id),
                gt(schema.keys.remaining, 0), // prevent negative remaining
              ),
            )
            .execute(),
        );
        // revalidate every minute
        if (Date.now() - this.lastRevalidate > 60_000) {
          this.logger.info("revalidating in the background", { keyId: this.key.id });
          this.state.waitUntil(
            this.db.query.keys
              .findFirst({
                where: (table, { and, eq, isNull }) =>
                  and(eq(table.id, req.keyId), isNull(table.deletedAt)),
              })
              .execute()
              .then((key) => {
                this.key = key;
                this.lastRevalidate = Date.now();
              }),
          );
        }

        return Response.json({
          valid: true,
          remaining: this.key.remaining,
        });
      }
    }
  }
}

import { type Database, type Key, and, createConnection, eq, gt, schema, sql } from "@/pkg/db";
import { ConsoleLogger } from "@unkey/worker-logging";
import type { Env } from "../env";
import { limitRequestSchema, revalidateRequestSchema } from "./interface";

export class DurableObjectUsagelimiter implements DurableObject {
  private readonly state: DurableObjectState;
  private readonly db: Database;
  private lastRevalidate = 0;
  private key: Key | undefined = undefined;
  private readonly env: Env;

  constructor(state: DurableObjectState, env: Env) {
    this.state = state;
    this.env = env;
    this.db = createConnection({
      host: env.DATABASE_HOST,
      password: env.DATABASE_PASSWORD,
      username: env.DATABASE_USERNAME,
      retry: false,
      logger: new ConsoleLogger({
        requestId: "",
        application: "api",
        environment: this.env.ENVIRONMENT,
      }),
    });
  }

  async fetch(request: Request) {
    const logger = new ConsoleLogger({
      application: "api",
      environment: this.env.ENVIRONMENT,
      requestId: request.headers.get("Unkey-Request-Id") ?? "",
      defaultFields: {
        durableObjectId: this.state.id.toString(),
        durableObjectClass: "DurableObjectUsagelimiter",
        environment: this.env.ENVIRONMENT,
      },
    });
    const url = new URL(request.url);
    switch (url.pathname) {
      case "/revalidate": {
        const req = revalidateRequestSchema.parse(await request.json());

        this.key = await this.db.query.keys.findFirst({
          where: (table, { and, eq, isNull }) =>
            and(eq(table.id, req.keyId), isNull(table.deletedAtM)),
        });
        this.lastRevalidate = Date.now();
        return Response.json({});
      }
      case "/limit": {
        const req = limitRequestSchema.parse(await request.json());
        if (!this.key) {
          this.key = await this.db.query.keys.findFirst({
            where: (table, { and, eq, isNull }) =>
              and(eq(table.id, req.keyId), isNull(table.deletedAtM)),
          });
          this.lastRevalidate = Date.now();
        }

        if (!this.key) {
          logger.error("key not found", { keyId: req.keyId });
          return Response.json({
            valid: false,
          });
        }

        if (this.key.remaining === null) {
          logger.warn("key does not have remaining requests enabled", {
            key: this.key,
          });
          return Response.json({
            valid: true,
          });
        }

        if (this.key.remaining <= 0 && req.cost !== 0) {
          return Response.json({
            valid: false,
            remaining: 0,
          });
        }

        this.key.remaining = Math.max(0, this.key.remaining - req.cost);

        this.state.waitUntil(
          this.db
            .update(schema.keys)
            .set({ remaining: sql`${schema.keys.remaining}-${req.cost}` })
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
          logger.info("revalidating in the background", { keyId: this.key.id });
          this.state.waitUntil(
            this.db.query.keys
              .findFirst({
                where: (table, { and, eq, isNull }) =>
                  and(eq(table.id, req.keyId), isNull(table.deletedAtM)),
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
    return new Response("invalid path", { status: 404 });
  }
}
